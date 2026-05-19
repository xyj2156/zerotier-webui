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
        [
            'success' => $success,
            'result'  => $result,
            'message' => $message,
        ] = $this->zt->getStatus();

        return $this->json(
            $success ? 0 : 1,
            $message,
            $result,
        );
    }

    // 网络列表
    public function networks()
    {
        [
            'success' => $success, 'result' => $result, 'message' => $message,
        ] = $this->zt->networks();
        return $this->json(
            $success ? 0 : 1,
            $message,
            $result,
        );
    }

    public function networkCount()
    {
        [
            'success' => $success, 'result' => $result, 'message' => $message,
        ] = $this->zt->networksCount();

        return $this->json(
            $success ? 0 : 1,
            $message,
            $result,
        );
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
        $response = $this->zt->getNetwork($nwid);
        if ($response['success']) {
            return $this->json(
                0,
                $response['message'],
                $response['result'],
            );
        }
        return $this->json(
            1,
            $response['message'],
            $response['result'],
        );
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

    protected function json(int $status = 0, string|array $message = '', iterable|int|string|null $result = null)
    {
        return [
            'status'  => $status,
            'message' => $message,
            'result'  => $result,
        ];
    }
}
