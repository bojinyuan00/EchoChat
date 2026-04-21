/**
 * 测试专用守卫：仅允许 NODE_ENV=test 调用
 *
 * 所有下划线前缀的 _clearXxxMap / __TEST__* 等测试辅助函数必须在函数体首行调用
 * `assertTestOnly('<caller>')`，防止被误用到生产路径，造成资源全量销毁。
 */
export function assertTestOnly(caller: string): void {
  if (process.env.NODE_ENV !== 'test') {
    throw new Error(
      `[test-guard] ${caller} is test-only and must not be called in production (NODE_ENV=${process.env.NODE_ENV ?? 'undefined'})`,
    );
  }
}
