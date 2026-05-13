<?php

namespace App\Http\Controllers;

use App\Services\ZerotierService;
use Illuminate\Http\Request;
use Random\RandomException;

/**
 * class ZerotierController
 *
 * @package App\Http\Controllers
 */
class ZerotierController
{
    protected ZerotierService $zt;

    public function __construct(ZerotierService $zt)
    {
        $this->zt = $zt;
    }

    // 仪表盘状态
    public function status()
    {
        return $this->zt->getStatus();
    }

    // 网络列表
    public function networks()
    {
        return $this->zt->networks();
    }

    /**
     * 创建网络
     *
     * @param Request $request
     *
     * @return array
     * @throws RandomException
     */
    public function createNetwork(Request $request)
    {
        return $this->zt->createNetwork($request->name);
    }

    // 网络详情
    public function network($nwid)
    {
        return $this->zt->getNetwork($nwid);
    }

    // 成员列表
    public function members($nwid)
    {
        return $this->zt->getMembers($nwid);
    }

    // 更新成员
    public function updateMember(Request $request, $nwid, $id)
    {
        return $this->zt->updateMember($nwid, $id, $request->all());
    }

    // 删除成员
    public function deleteMember($nwid, $id)
    {
        return $this->zt->deleteMember($nwid, $id);
    }
}
