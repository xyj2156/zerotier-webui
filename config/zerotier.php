<?php

return [

    /*
    |--------------------------------------------------------------------------
    | ZeroTier 本地自建控制器 API（zerotier-one）
    |--------------------------------------------------------------------------
    |
    | 契约：zerotier-one 内置控制器的本地 REST API（非官方 Central）。
    |
    | base_url：默认本机 9993；若控制器在别的机器/端口，改此值即可。
    | token：控制器所在机器 ~/.zerotier/authtoken_secret 文件内容，
    |         请求头以 `X-ZT1-Auth: <TOKEN>` 发送。
    | node_address：控制器自身 10 位节点地址（可选）。留空时服务会调用
    |         GET /status 自动获取，用于拼新建网络的 16 位 nwid（前 10 位
    |         必须等于控制器节点地址，否则本地控制器拒绝创建）。
    |
    */

    'base_url'     => env('ZEROTIER_API_URL') ?: 'http://127.0.0.1:9993',

    'token'        => env('ZEROTIER_API_TOKEN'),

    'node_address' => env('ZEROTIER_NODE_ADDRESS'),

    'timeout'      => (int) (env('ZEROTIER_TIMEOUT') ?: 15),

    // 新建网络默认模板（会被前端提交字段整体覆盖/合并）
    'default_network' => [
        'active'          => true,
        'private'         => true,
        'enableBroadcast' => true,
        'mtu'             => 2800,
        'multicastTTL'    => 128,
        'multicastLimit'  => 32,
        'v4AssignMode'    => ['zt' => true],
        'v6AssignMode'    => ['6plane' => false, 'rfc4193' => false, 'zt' => false],
        'routes'          => [],
        'ipAssignmentPools' => [],
    ],
];
