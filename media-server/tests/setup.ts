process.env.NODE_ENV = 'test';
process.env.MEDIA_INTERNAL_TOKEN = process.env.MEDIA_INTERNAL_TOKEN ?? 'test-token-1234567890';
process.env.LOG_LEVEL = process.env.LOG_LEVEL ?? 'silent';
process.env.LOG_PRETTY = process.env.LOG_PRETTY ?? 'false';
process.env.MEDIASOUP_RTC_MIN_PORT = process.env.MEDIASOUP_RTC_MIN_PORT ?? '40800';
process.env.MEDIASOUP_RTC_MAX_PORT = process.env.MEDIASOUP_RTC_MAX_PORT ?? '40899';
process.env.MEDIASOUP_WORKER_LOG_LEVEL = process.env.MEDIASOUP_WORKER_LOG_LEVEL ?? 'error';
