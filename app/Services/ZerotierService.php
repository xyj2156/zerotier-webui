<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;
use Random\RandomException;

class ZeroTierService
{
    protected string $baseUrl;
    protected string $token;

    /**
     * @throws \Exception
     */
    public function __construct()
    {
        // Zerotier 本地 API 地址 + 令牌
        $this->baseUrl = config('zerotier.api_url');
        $this->token   = config('zerotier.api_token');
        if (empty($this->baseUrl) || empty($this->token)) {
            throw new \Exception('Zerotier API 地址或令牌未配置');
        }
    }

    /**
     * 发送请求的底层方法
     */
    protected function request(string $method, string $uri, array $data = []): array
    {
        try {
            $response = Http::withHeaders([
                'X-ZT1-Auth'   => $this->token,
                'Content-Type' => 'application/json',
            ])->$method("{$this->baseUrl}{$uri}", $data);

            return [
                'success' => $response->successful(),
                'status'  => $response->status(),
                'data'    => $response->json(),
            ];
        } catch (\Exception $e) {
            return ['success' => false, 'error' => $e->getMessage()];
        }
    }

    public function getStatus(): array
    {
        return $this->request('get', '/status');
    }

    // ================= 网络管理 =================

    /**
     * 获取所有网络列表
     */
    public function networks(): array
    {
        $list = $this->request('get', '/controller/network');
        foreach ($list['data'] as &$item) {
            $temp = $this->getNetwork($item)['data'] ?? null;
            $item = [
                'id'      => $temp['id'],
                'nwid'    => $temp['nwid'],
                'private' => $temp['private'],
                'name'    => $temp['name'],
            ];
        }
        return ['success' => true, 'data' => $list['data']];
    }

    /**
     * 创建一个新网络
     *
     * @throws RandomException
     */
    public function createNetwork(string $name): array
    {
        // 随机生成一个16位的网络ID
        $networkId = bin2hex(random_bytes(8));

        // 默认配置：私有网络，自动分配IP
        $defaultConfig = [
            'name'              => $name,
            'private'           => true,
            'enableBroadcast'   => true,
            'ipAssignmentPools' => [
                ['ipRangeStart' => '10.147.20.1', 'ipRangeEnd' => '10.147.20.254'],
            ],
            'routes'            => [
                ['target' => '10.147.20.0/24', 'via' => null],
            ],
            'v4AssignMode'      => ['zt' => true],
        ];

        return $this->request('post', "/controller/network/{$networkId}", $defaultConfig);
    }

    /**
     * 删除网络
     */
    public function deleteNetwork(string $networkId): array
    {
        return $this->request('delete', "/controller/network/{$networkId}");
    }

    /**
     * 获取网络详情（包含所有成员）
     */
    public function getNetwork(string $networkId): array
    {
        return $this->request('get', "/controller/network/{$networkId}");
    }

    /**
     * 更新网络配置（如修改名称、网段）
     */
    public function updateNetwork(string $networkId, array $config): array
    {
        return $this->request('post', "/controller/network/{$networkId}", $config);
    }

    // ================= 成员与授权管理 =================

    /**
     * 获取某个网络下的所有成员
     */
    public function getMembers(string $networkId): array
    {
        return $this->request('get', "/controller/network/{$networkId}/member");
    }

    /**
     * 获取指定成员的详情
     */
    public function getMember(string $networkId, string $nodeId): array
    {
        return $this->request('get', "/controller/network/{$networkId}/member/{$nodeId}");
    }

    /**
     * 授权或取消授权成员
     */
    public function authorizeMember(string $networkId, string $nodeId, bool $authorized = true): array
    {
        return $this->request('post', "/controller/network/{$networkId}/member/{$nodeId}", [
            'authorized' => $authorized,
        ]);
    }

    /**
     * 删除成员（踢出网络）
     */
    public function deleteMember(string $networkId, string $nodeId): array
    {
        return $this->request('delete', "/controller/network/{$networkId}/member/{$nodeId}");
    }

    /**
     * 为成员分配固定IP (高级功能)
     */
    public function assignIp(string $networkId, string $nodeId, array $ips): array
    {
        return $this->request('post', "/controller/network/{$networkId}/member/{$nodeId}", [
            'ipAssignments' => $ips,
        ]);
    }
}
