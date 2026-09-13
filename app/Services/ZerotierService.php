<?php

namespace App\Services;

use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\Http;
use Random\RandomException;

/**
 * ZeroTier 本地自建控制器（zerotier-one）API 客户端。
 *
 * 契约：zerotier-one 内置控制器的本地 REST API（端口默认 9993）。
 * 鉴权：请求头 `X-ZT1-Auth: <authtoken_secret>`。
 *
 * 与官方 Central 的关键差异已在此屏蔽：
 *  - 列表 GET /controller/network 只返回 nwid 字符串数组，需逐个取 config；
 *  - 新建网络 nwid 必须 = 控制器节点地址(10hex) + 6 位随机 hex；
 *  - 没有 /route、/node、/peer drop 等独立端点，路由在网络 config 内编辑；
 *  - network config / member 走「整体透传」，前端给什么可写字段就写什么，
 *    仅剥离服务端只读字段，以最大化可配置面。
 *
 * 统一返回信封：['success' => bool, 'result' => mixed, 'message' => string]
 *
 * class ZerotierService
 *
 * @package App\Services
 */
class ZerotierService
{
    protected string $baseUrl;
    protected string $token;
    protected int $timeout;
    protected ?string $nodeAddress = null;

    /** 成员里的服务端只读字段，回写前剥离 */
    protected const MEMBER_READONLY = [
        'id', 'address', 'nodeId', 'networkId', 'creationTime', 'lastOnline',
        'lastActive', 'online', 'physicalAddress', 'publicAddress',
        'clientVersion', 'identity', 'allowedIps', 'updatedAt', 'errors', 'latency',
    ];

    /** 网络 config 里的只读字段，回写前剥离（id/nwid 由服务端注入） */
    protected const NETWORK_READONLY = [
        'id', 'nwid', 'creationTime', 'lastModified', 'revision', 'stats',
    ];

    /**
     * @throws \Exception
     */
    public function __construct()
    {
        $this->baseUrl = rtrim((string) config('zerotier.base_url'), '/');
        $this->token   = (string) config('zerotier.token');
        $this->timeout = (int) (config('zerotier.timeout') ?: 15);

        if ($this->baseUrl === '' || $this->token === '') {
            throw new \Exception('ZeroTier 控制器 API 地址或令牌未配置');
        }
    }

    // ================= 底层请求 =================

    /**
     * @return array{success: bool, result: mixed, message: string}
     */
    protected function request(string $method, string $uri, array $data = [], array $query = []): array
    {
        try {
            $response = Http::withHeaders(['X-ZT1-Auth' => $this->token])
                ->acceptJson()
                ->timeout($this->timeout)
                ->$method("{$this->baseUrl}{$uri}", $query !== [] ? $query : $data);

            return $this->normalize($response);
        } catch (\Throwable $e) {
            return ['success' => false, 'result' => null, 'message' => $e->getMessage()];
        }
    }

    protected function get(string $uri, array $query = []): array
    {
        return $this->request('get', $uri, [], $query);
    }

    protected function post(string $uri, array $data = [], array $query = []): array
    {
        return $this->request('post', $uri, $data, $query);
    }

    protected function delete(string $uri, array $query = []): array
    {
        return $this->request('delete', $uri, [], $query);
    }

    protected function normalize(Response $response): array
    {
        if ($response->successful()) {
            $json = $response->json();

            return ['success' => true, 'result' => $json === null ? true : $json, 'message' => ''];
        }

        $message = $response->json('message') ?: ($response->body() !== '' ? $response->body() : "HTTP {$response->status()}");

        return ['success' => false, 'result' => null, 'message' => (string) $message];
    }

    // ================= 状态 =================

    public function getStatus(): array
    {
        return $this->get('/status');
    }

    /**
     * 解析控制器节点地址（新建网络 nwid 前缀用）。缓存于实例。
     */
    protected function nodeAddress(): ?string
    {
        if ($this->nodeAddress !== null) {
            return $this->nodeAddress;
        }

        $configured = (string) config('zerotier.node_address');
        if ($configured !== '') {
            return $this->nodeAddress = strtolower($configured);
        }

        $status = $this->get('/status');
        $addr   = $status['result']['address'] ?? null;

        return $this->nodeAddress = $addr ? strtolower((string) $addr) : null;
    }

    // ================= 网络管理 =================

    /**
     * 网络列表：GET /controller/network 返回 nwid 数组，逐个补 config 关键字段。
     */
    public function networks(): array
    {
        $res = $this->get('/controller/network');
        if (!$res['success']) {
            return $res;
        }

        $list = [];
        foreach ((array) $res['result'] as $nwid) {
            $detail = $this->getNetwork((string) $nwid);
            if (!$detail['success']) {
                // 详情取不到也至少给出 id
                $list[] = ['id' => $nwid, 'nwid' => $nwid, 'name' => '', 'private' => null, 'error' => $detail['message']];
                continue;
            }
            $c      = (array) $detail['result'];
            $list[] = [
                'id'           => $c['id'] ?? $nwid,
                'nwid'         => $c['nwid'] ?? $nwid,
                'name'         => $c['name'] ?? '',
                'private'      => (bool) ($c['private'] ?? true),
                'active'       => (bool) ($c['active'] ?? true),
                'creationTime' => $c['creationTime'] ?? null,
            ];
        }

        return ['success' => true, 'result' => $list, 'message' => ''];
    }

    public function networksCount(): array
    {
        $res = $this->get('/controller/network');
        if (!$res['success']) {
            return $res;
        }

        return ['success' => true, 'result' => count((array) $res['result']), 'message' => ''];
    }

    /**
     * 网络详情（完整 config），GET /controller/network/{nwid}。
     */
    public function getNetwork(string $nwid): array
    {
        return $this->get("/controller/network/{$nwid}");
    }

    /**
     * 创建网络：本地 API 需 POST /controller/network/{nwid}，
     * nwid = 控制器节点地址(10hex) + 6 位随机 hex；缺省字段用模板补全，其余透传。
     *
     * @throws RandomException
     */
    public function createNetwork(array $config): array
    {
        $address = $this->nodeAddress();
        if (!$address || strlen($address) !== 10) {
            return ['success' => false, 'result' => null, 'message' => '无法获取控制器节点地址，不能生成合法 nwid（可在 .env 配置 ZEROTIER_NODE_ADDRESS）'];
        }

        $nwid   = $address . bin2hex(random_bytes(3));
        $payload = array_merge((array) config('zerotier.default_network'), $config);
        foreach (self::NETWORK_READONLY as $k) {
            unset($payload[$k]);
        }
        $payload['id']   = $nwid;
        $payload['nwid'] = $nwid;
        $payload['active'] = $payload['active'] ?? true;

        return $this->post("/controller/network/{$nwid}", $payload);
    }

    /**
     * 更新网络配置：POST /controller/network/{nwid}，透传前端给的完整 config。
     */
    public function updateNetwork(string $nwid, array $config): array
    {
        foreach (self::NETWORK_READONLY as $k) {
            unset($config[$k]);
        }
        $config['id']   = $nwid;
        $config['nwid'] = $nwid;

        return $this->post("/controller/network/{$nwid}", $config);
    }

    public function deleteNetwork(string $nwid): array
    {
        return $this->delete("/controller/network/{$nwid}");
    }

    // ================= 成员管理 =================

    /**
     * 某网络全部成员。本地返回以地址为键的对象映射，统一规整成数组并补 nodeId。
     */
    public function getMembers(string $nwid): array
    {
        $res = $this->get("/controller/network/{$nwid}/member");
        if (!$res['success']) {
            return $res;
        }

        $raw   = (array) $res['result'];
        $list  = [];
        foreach ($raw as $key => $m) {
            if (!is_array($m)) {
                continue; // 旧版可能返回地址数组
            }
            $m['nodeId'] ??= is_string($key) ? $key : ($m['address'] ?? '');
            $list[] = $m;
        }

        return ['success' => true, 'result' => $list, 'message' => ''];
    }

    public function getMember(string $nwid, string $nodeId): array
    {
        return $this->get("/controller/network/{$nwid}/member/{$nodeId}");
    }

    /**
     * 更新成员：POST 透传任意可写字段，剥离只读字段。
     */
    public function updateMember(string $nwid, string $nodeId, array $member): array
    {
        foreach (self::MEMBER_READONLY as $k) {
            unset($member[$k]);
        }

        return $this->post("/controller/network/{$nwid}/member/{$nodeId}", $member);
    }

    public function authorizeMember(string $nwid, string $nodeId, bool $authorized = true): array
    {
        return $this->post("/controller/network/{$nwid}/member/{$nodeId}", ['authorized' => $authorized]);
    }

    public function deleteMember(string $nwid, string $nodeId): array
    {
        return $this->delete("/controller/network/{$nwid}/member/{$nodeId}");
    }

    // ================= 通用透传 =================

    /**
     * 只读透传：GET 任意本地控制器路径，供前端高级视图访问未封装的端点。
     */
    public function rawGet(string $uri): array
    {
        return $this->get('/' . ltrim($uri, '/'));
    }

    /**
     * 写入透传：POST 任意本地控制器路径（如 /admin 命令）。
     */
    public function rawPost(string $uri, array $data = []): array
    {
        return $this->post('/' . ltrim($uri, '/'), $data);
    }
}
