<?php

namespace App\Services;

use Illuminate\Support\Facades\Hash;
use RuntimeException;

/**
 * 极简 JWT（HS256）签发与校验，零外部依赖。
 *
 * class JwtService
 *
 * @package App\Services
 */
class JwtService
{
    protected string $secret;
    protected int $ttl;
    protected string $issuer;

    public function __construct()
    {
        $this->secret = (string) config('jwt.secret');
        $this->ttl    = (int) config('jwt.ttl');
        $this->issuer = (string) config('jwt.issuer');

        if ($this->secret === '') {
            throw new RuntimeException('JWT 密钥未配置（JWT_SECRET / APP_KEY 均为空）');
        }
    }

    /**
     * 为给定用户签发访问令牌。
     */
    public function issue(mixed $userId, array $extra = []): string
    {
        $now = time();
        $payload = array_merge([
            'iss' => $this->issuer,
            'sub' => (string) $userId,
            'iat' => $now,
            'exp' => $now + $this->ttl,
        ], $extra);

        $segments   = [
            $this->b64((string) json_encode(['alg' => 'HS256', 'typ' => 'JWT'])),
            $this->b64((string) json_encode($payload)),
        ];
        $segments[] = $this->b64($this->sign(implode('.', $segments)));

        return implode('.', $segments);
    }

    /**
     * 校验并返回 payload；非法/过期抛 RuntimeException。
     *
     * @return array<string, mixed>
     */
    public function verify(string $jwt): array
    {
        $parts = explode('.', $jwt);
        if (count($parts) !== 3) {
            throw new RuntimeException('令牌格式错误');
        }
        [$h, $p, $s] = $parts;

        $expected = $this->b64($this->sign("{$h}.{$p}"));
        if (!hash_equals($expected, $s)) {
            throw new RuntimeException('签名校验失败');
        }

        $payload = json_decode($this->unb64($p), true);
        if (!is_array($payload)) {
            throw new RuntimeException('令牌负载解析失败');
        }
        if (($payload['exp'] ?? 0) < time()) {
            throw new RuntimeException('令牌已过期');
        }

        return $payload;
    }

    /**
     * 恒定时间比对明文密码与哈希。
     */
    public static function checkPassword(string $plain, string $hash): bool
    {
        return Hash::check($plain, $hash);
    }

    protected function sign(string $data): string
    {
        return hash_hmac('sha256', $data, $this->secret, true);
    }

    protected function b64(string $data): string
    {
        return rtrim(strtr(base64_encode($data), '+/', '-_'), '=');
    }

    protected function unb64(string $data): string
    {
        $decoded = base64_decode(strtr($data, '-_', '+/'), true);

        return $decoded === false ? '' : $decoded;
    }
}
