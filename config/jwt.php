<?php

return [

    // 签名密钥；为空时回退到 APP_KEY，避免遗漏配置导致无法签发
    'secret' => env('JWT_SECRET') ?: (string) env('APP_KEY'),

    // 令牌有效期（秒）
    'ttl' => (int) (env('JWT_TTL') ?: 7200),

    // 签发方标识
    'issuer' => env('JWT_ISSUER', 'zerotier-webui'),
];
