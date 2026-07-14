import { networkInterfaces, type NetworkInterfaceInfo } from 'node:os';

import { childLogger } from './logger.js';

const log = childLogger({ module: 'utils.network' });

/**
 * 候选网卡 IPv4 信息：包含网卡名 + 地址 + RFC1918 段标签。
 * 段标签用于"优先级排序"：192.168 / 10. / 172.16-31. 三类私有段。
 */
export interface LanIpv4Candidate {
  /** 网卡名，例如 en0 / eth0 / wlan0 */
  ifaceName: string;
  /** IPv4 地址 */
  address: string;
  /** RFC1918 段类别，用于优先级排序 */
  rank: 'private-192' | 'private-10' | 'private-172' | 'unknown';
}

/**
 * 已知的虚拟网卡前缀黑名单：Parallels / VMware / Docker bridge / TUN 等。
 * 这些网卡上的 IP 通常无法被同 LAN 内的物理设备路由到，必须排除掉。
 */
const VIRTUAL_IFACE_PREFIXES = [
  'lo',
  'docker',
  'br-',
  'veth',
  'utun',
  'awdl',
  'llw',
  'gif',
  'stf',
  'anpi',
  'ap',
  'bridge',
  'vmnet',
  'vnic',
  'vboxnet',
  'tun',
  'tap',
];

function isVirtualIface(name: string): boolean {
  const lower = name.toLowerCase();
  return VIRTUAL_IFACE_PREFIXES.some((p) => lower.startsWith(p));
}

function classifyPrivate(ip: string): LanIpv4Candidate['rank'] {
  if (ip.startsWith('192.168.')) return 'private-192';
  if (ip.startsWith('10.')) return 'private-10';
  if (ip.startsWith('172.')) {
    const second = Number.parseInt(ip.split('.')[1] ?? '', 10);
    if (Number.isFinite(second) && second >= 16 && second <= 31) {
      return 'private-172';
    }
  }
  return 'unknown';
}

/**
 * 列出本机所有"非虚拟、非 internal、非 link-local 的 IPv4 网卡候选"。
 *
 * 排序规则（优先级从高到低）：
 *   1. 192.168.x.x（家用路由器最常见，跨设备最稳）
 *   2. 10.x.x.x（企业内网常见）
 *   3. 172.16-31.x.x（容器/VPN 常见，但物理 LAN 也合法）
 */
export function listLanIpv4Interfaces(): LanIpv4Candidate[] {
  const all = networkInterfaces();
  const candidates: LanIpv4Candidate[] = [];

  for (const [ifaceName, addrs] of Object.entries(all)) {
    if (isVirtualIface(ifaceName)) continue;
    if (!addrs) continue;

    for (const a of addrs as NetworkInterfaceInfo[]) {
      if (a.family !== 'IPv4') continue;
      if (a.internal) continue;
      // 169.254.x.x 是 link-local，物理 LAN 几乎用不到
      if (a.address.startsWith('169.254.')) continue;

      candidates.push({
        ifaceName,
        address: a.address,
        rank: classifyPrivate(a.address),
      });
    }
  }

  const order: Record<LanIpv4Candidate['rank'], number> = {
    'private-192': 0,
    'private-10': 1,
    'private-172': 2,
    unknown: 3,
  };
  candidates.sort((a, b) => order[a.rank] - order[b.rank]);
  return candidates;
}

/**
 * 自动探测一个最适合作为 mediasoup `announcedIp` 的本机内网 IPv4。
 * 失败返回 null（生产环境通常不应到达此分支，因为 ANNOUNCED_IP 必填公网 IP）。
 */
export function detectLanIpv4(): string | null {
  const candidates = listLanIpv4Interfaces();
  return candidates[0]?.address ?? null;
}

/**
 * 判断给定 IP 是否落在 RFC1918 三大私网段。
 * 公网 IP（包括云厂商 EIP）会返回 false。
 */
function isPrivateIpv4(ip: string): boolean {
  return classifyPrivate(ip) !== 'unknown';
}

/**
 * 解析 mediasoup `announcedIp` 的最终生效值。
 *
 * 决策矩阵：
 *   1. configured 非空 + 命中本机网卡 → 直接用 configured（最常见路径）
 *   2. configured 非空 + 是公网 IP（非 RFC1918）→ 直接用 configured（公网部署 EIP/NAT 场景）
 *   3. configured 非空 + 是私网 IP 但不在本机网卡 + 非生产 → **自动覆盖为当前探测到的 LAN IP**
 *      （dev 场景下这几乎可以肯定是 Wi-Fi/局域网漂移导致的过期值；之前只 warn 不覆盖会让用户反复手改 .env）
 *   4. configured 非空 + 是私网 IP 但不在本机网卡 + 生产 → 用 configured（生产强信任，仅 WARN）
 *   5. configured 空 + 非生产 → 自动探测本机活跃内网 IP
 *   6. configured 空 + 生产 → 返回 null 并 ERROR 提示必须显式配置
 *
 * @param configured 来自 env `MEDIASOUP_ANNOUNCED_IP` 的原始值，留空时为 undefined
 * @param isProduction NODE_ENV === 'production'
 * @returns 最终用于 mediasoup `webRtcTransportOptions.listenIps[].announcedIp` 的字符串；null 表示不传该字段
 */
export function resolveAnnouncedIp(
  configured: string | undefined,
  isProduction: boolean,
): string | null {
  if (configured && configured.length > 0) {
    const candidates = listLanIpv4Interfaces();
    const matched = candidates.some((c) => c.address === configured);
    if (matched) {
      return configured;
    }

    // 公网 IP（非 RFC1918 私网段）：通常是 EIP / NAT / ngrok，本机网卡不会有，强信任
    if (!isPrivateIpv4(configured)) {
      log.info(
        { announcedIp: configured },
        'configured MEDIASOUP_ANNOUNCED_IP looks like a public IP (EIP/NAT) — trusted as-is',
      );
      return configured;
    }

    // 私网 IP 但不在本机网卡：dev 模式下视为"残留过期值"，自动覆盖为 detected
    if (!isProduction) {
      const detected = detectLanIpv4();
      if (detected) {
        log.warn(
          {
            stale: configured,
            override: detected,
            candidates: candidates.map((c) => `${c.ifaceName}=${c.address}`),
          },
          'configured MEDIASOUP_ANNOUNCED_IP is a private IP not bound to any local interface '
            + '— treating as stale (Wi-Fi/LAN drift) and AUTO-OVERRIDING to detected LAN IP. '
            + 'To silence this, update or empty MEDIASOUP_ANNOUNCED_IP in media-server/.env',
        );
        return detected;
      }
      // 探不到任何活跃 LAN，只能仍用 configured 兜底，至少不阻断启动
      log.warn(
        {
          announcedIp: configured,
          localCandidates: candidates.map((c) => `${c.ifaceName}=${c.address}`),
        },
        'configured MEDIASOUP_ANNOUNCED_IP not present on any local interface AND '
          + 'no active LAN IPv4 detected — falling back to configured value, ICE will likely fail',
      );
      return configured;
    }

    // 生产环境且 configured 是私网 IP 但不命中本机：可能是容器内网/K8s Pod IP，仅 WARN 不覆盖
    log.warn(
      {
        announcedIp: configured,
        localCandidates: candidates.map((c) => `${c.ifaceName}=${c.address}`),
      },
      'configured MEDIASOUP_ANNOUNCED_IP not present on any local interface in production — '
        + 'this may be intentional (containerized deployment with port forwarding), '
        + 'but verify NAT/Hairpin rules if WebRTC ICE fails',
    );
    return configured;
  }

  if (isProduction) {
    log.error(
      'MEDIASOUP_ANNOUNCED_IP is required in production but is empty; '
        + 'WebRTC ICE will fail for all remote peers. Set it to your public IP / EIP.',
    );
    return null;
  }

  const detected = detectLanIpv4();
  if (detected) {
    const candidates = listLanIpv4Interfaces();
    log.info(
      {
        detected,
        candidates: candidates.map((c) => `${c.ifaceName}=${c.address}`),
      },
      'MEDIASOUP_ANNOUNCED_IP not configured, auto-detected local LAN IPv4',
    );
    return detected;
  }

  log.warn(
    'MEDIASOUP_ANNOUNCED_IP is empty and no usable LAN IPv4 detected; '
      + 'WebRTC ICE will likely fail. Configure MEDIASOUP_ANNOUNCED_IP explicitly.',
  );
  return null;
}
