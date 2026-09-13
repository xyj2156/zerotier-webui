<?php

namespace App\Http\Controllers;

use App\Services\ZerotierService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Random\RandomException;

/**
 * class ZerotierController
 *
 * 统一响应信封：{ status:int(0=成功), message:string, result:mixed }
 *
 * @package App\Http\Controllers
 */
class ZerotierController extends Controller
{
    public function __construct(protected ZerotierService $zt)
    {
    }

    // ================= 状态 =================

    public function status(): JsonResponse
    {
        return $this->envelope($this->zt->getStatus());
    }

    // ================= 网络 =================

    public function networks(): JsonResponse
    {
        return $this->envelope($this->zt->networks());
    }

    public function networkCount(): JsonResponse
    {
        return $this->envelope($this->zt->networksCount());
    }

    public function network(string $nwid): JsonResponse
    {
        return $this->envelope($this->zt->getNetwork($nwid));
    }

    /**
     * 创建网络：透传完整配置，type/cidr 为便捷字段。
     *
     * @throws RandomException
     */
    public function createNetwork(Request $request): JsonResponse
    {
        return $this->envelope($this->zt->createNetwork($this->normalizeNetworkInput($request)));
    }

    public function updateNetwork(Request $request, string $nwid): JsonResponse
    {
        return $this->envelope($this->zt->updateNetwork($nwid, $this->normalizeNetworkInput($request)));
    }

    public function deleteNetwork(string $nwid): JsonResponse
    {
        return $this->envelope($this->zt->deleteNetwork($nwid));
    }

    // ================= 成员 =================

    public function members(string $nwid): JsonResponse
    {
        return $this->envelope($this->zt->getMembers($nwid));
    }

    public function member(string $nwid, string $id): JsonResponse
    {
        return $this->envelope($this->zt->getMember($nwid, $id));
    }

    public function updateMember(Request $request, string $nwid, string $id): JsonResponse
    {
        return $this->envelope($this->zt->updateMember($nwid, $id, $request->all()));
    }

    public function deleteMember(string $nwid, string $id): JsonResponse
    {
        return $this->envelope($this->zt->deleteMember($nwid, $id));
    }

    // ================= 通用透传（高级） =================

    /**
     * GET 透传：/api/raw?path=/controller/network/{nwid} 等，访问未封装端点。
     */
    public function raw(Request $request): JsonResponse
    {
        $validated = $request->validate(['path' => ['required', 'string']]);

        return $this->envelope($this->zt->rawGet($validated['path']));
    }

    // ================= 内部工具 =================

    /**
     * 归一化网络写入负载：整体透传（尽量保留全部可配置字段），仅处理
     *  - type: 'private'|'public' → private 布尔
     *  - cidr: 便捷字段，未显式给出 pools/routes 时展开
     */
    protected function normalizeNetworkInput(Request $request): array
    {
        $request->validate([
            'private' => ['sometimes', 'boolean'],
            'type'    => ['sometimes', 'in:private,public'],
        ]);

        // 透传除便捷字段外的全部内容
        $payload = $request->except(['type', 'cidr', '_token']);

        if ($request->has('type')) {
            $payload['private'] = $request->string('type')->toString() === 'private';
        }

        $cidr = $request->string('cidr')->toString();
        if ($cidr !== '' && empty($payload['ipAssignmentPools']) && empty($payload['routes'])) {
            $pool = $this->cidrToPool($cidr);
            if ($pool !== null) {
                $payload['ipAssignmentPools'] = [[
                    'ipRangeStart' => $pool['start'],
                    'ipRangeEnd'   => $pool['end'],
                ]];
                $payload['routes'] = [[
                    'target' => $pool['network'] . '/' . $pool['prefix'],
                    'via'    => null,
                ]];
            }
        }

        return $payload;
    }

    /**
     * @return array{network:string,prefix:int,start:string,end:string}|null
     */
    protected function cidrToPool(string $cidr): ?array
    {
        if (count($parts = explode('/', trim($cidr))) !== 2) {
            return null;
        }
        [$ip, $prefix] = $parts;
        $prefix = (int) $prefix;
        $ip32   = ip2long($ip);
        if ($ip32 === false || $prefix < 1 || $prefix > 31) {
            return null;
        }

        $host  = (1 << (32 - $prefix)) - 1;
        $net32 = $ip32 & ~$host;
        $bcast = $net32 | $host;

        return [
            'network' => long2ip($net32),
            'prefix'  => $prefix,
            'start'   => long2ip($net32 + 1),
            'end'     => long2ip($bcast - 1),
        ];
    }

    /**
     * @param array{success: bool, result: mixed, message: string} $r
     */
    protected function envelope(array $r): JsonResponse
    {
        return response()->json([
            'status'  => $r['success'] ? 0 : 1,
            'message' => $r['message'] ?? '',
            'result'  => $r['result'] ?? null,
        ]);
    }
}
