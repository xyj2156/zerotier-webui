<?php

namespace App\Http\Controllers;

use App\Models\User;
use App\Services\JwtService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Hash;

/**
 * class LoginController
 *
 * 最小 JWT 登录：账号密码换发令牌，不引入注册/找回密码。
 *
 * @package App\Http\Controllers
 */
class LoginController extends Controller
{
    public function __construct(protected JwtService $jwt)
    {
    }

    /**
     * POST /api/login  { email, password }
     */
    public function login(Request $request): JsonResponse
    {
        $credentials = $request->validate([
            'email'    => ['required', 'email'],
            'password' => ['required', 'string'],
        ]);

        $user = User::where('email', $credentials['email'])->first();

        if (!$user || !Hash::check($credentials['password'], $user->password)) {
            return $this->fail('邮箱或密码错误');
        }

        $token = $this->jwt->issue($user->id, ['email' => $user->email]);

        return $this->ok([
            'token' => $token,
            'user'  => [
                'id'    => $user->id,
                'name'  => $user->name,
                'email' => $user->email,
            ],
        ]);
    }

    /**
     * GET /api/me —— 当前登录用户（由 Jwt 中间件注入 user resolver）
     */
    public function me(Request $request): JsonResponse
    {
        $user = $request->user();

        return $this->ok($user === null ? null : [
            'id'    => $user->id,
            'name'  => $user->name,
            'email' => $user->email,
        ]);
    }

    /**
     * POST /api/logout —— 无状态令牌，客户端丢弃即可；保留端点便于前端统一处理。
     */
    public function logout(): JsonResponse
    {
        return $this->ok(true, '已退出');
    }

    protected function ok(mixed $result, string $message = ''): JsonResponse
    {
        return response()->json(['status' => 0, 'message' => $message, 'result' => $result]);
    }

    protected function fail(string $message, int $code = 1): JsonResponse
    {
        return response()->json(['status' => $code, 'message' => $message, 'result' => null]);
    }
}
