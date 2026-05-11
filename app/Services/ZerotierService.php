<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;

/**
 * class ZerotierService
 *
 * @package App\Services
 */
class ZerotierService
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

    // 通用请求
    protected function request($method, $endpoint, $data = [])
    {
        return Http::withToken($this->token)->withHeaders([
            'Content-Type' => 'application/json',
            'X-ZT1-Auth'   => $this->token,
        ])->$method($this->baseUrl . $endpoint, $data);
    }

    // 获取状态
    public function getStatus()
    {
        return $this->request('get', '/status')->json();
    }

    // 获取所有网络
    public function getNetworks()
    {
        return $this->request('get', '/network')->json();
    }

    // 创建网络
    public function createNetwork($name, $subnet = '10.144.0.0/16')
    {
        return $this->request('post', '/network', [
            'name'    => $name,
            'private' => true,
            'routes'  => [['target' => $subnet, 'via' => null]],
        ])->json();
    }

    // 获取网络详情
    public function getNetwork($nwid)
    {
        return $this->request('get', "/network/$nwid")->json();
    }

    // 获取网络成员
    public function getMembers($nwid)
    {
        return $this->request('get', "/network/$nwid/member")->json();
    }

    // 修改成员（授权/改名/拉黑）
    public function updateMember($nwid, $memberId, $data)
    {
        return $this->request('post', "/network/$nwid/member/$memberId", $data)->json();
    }

    // 删除成员
    public function deleteMember($nwid, $memberId)
    {
        return $this->request('delete', "/network/$nwid/member/$memberId")->json();
    }
}
