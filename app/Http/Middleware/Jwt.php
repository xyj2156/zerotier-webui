<?php

namespace App\Http\Middleware;

use App\Models\User;
use App\Services\JwtService;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class Jwt
{
    public function __construct(protected JwtService $jwt)
    {
    }

    /**
     * 校验 `Authorization: Bearer <jwt>`；失败返回 401 统一信封。
     *
     * @param  Closure(Request): (Response)  $next
     */
    public function handle(Request $request, Closure $next): Response
    {
        $token = $request->bearerToken();

        if (!$token) {
            return $this->unauthorized('缺少访问令牌');
        }

        try {
            $payload = $this->jwt->verify($token);
        } catch (\Throwable $e) {
            return $this->unauthorized($e->getMessage());
        }

        $user = isset($payload['sub']) ? User::find($payload['sub']) : null;

        if (!$user) {
            return $this->unauthorized('用户不存在');
        }

        // 供控制器 $request->user() / attribute 读取
        $request->setUserResolver(static fn () => $user);
        $request->attributes->set('auth_user', $user);

        return $next($request);
    }

    protected function unauthorized(string $message): Response
    {
        return response()->json([
            'status'  => 401,
            'message' => $message,
            'result'  => null,
        ], 401);
    }
}
