<?php

namespace Tests\Feature;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AuthTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();
        config()->set('jwt.secret', 'unit-test-secret');
        config()->set('jwt.ttl', 3600);
    }

    protected function user(): User
    {
        return User::factory()->create([
            'email'    => 'admin@example.com',
            'password' => 'password',
        ]);
    }

    public function test_login_returns_token_and_user(): void
    {
        $this->user();

        $res = $this->postJson('/api/login', [
            'email'    => 'admin@example.com',
            'password' => 'password',
        ]);

        $res->assertOk()
            ->assertJsonPath('status', 0)
            ->assertJsonPath('result.user.email', 'admin@example.com');

        $this->assertNotEmpty($res->json('result.token'));
    }

    public function test_login_rejects_bad_credentials(): void
    {
        $this->user();

        $this->postJson('/api/login', [
            'email'    => 'admin@example.com',
            'password' => 'wrong',
        ])->assertOk()->assertJsonPath('status', 1);
    }

    public function test_protected_route_requires_token(): void
    {
        $this->getJson('/api/status')->assertStatus(401)->assertJsonPath('status', 401);
    }

    public function test_me_returns_current_user(): void
    {
        $user = $this->user();
        $token = app(\App\Services\JwtService::class)->issue($user->id);

        $this->withToken($token)
            ->getJson('/api/me')
            ->assertOk()
            ->assertJsonPath('result.email', 'admin@example.com');
    }

    public function test_rejects_tampered_token(): void
    {
        $user = $this->user();
        $token = app(\App\Services\JwtService::class)->issue($user->id);

        $this->withToken($token . 'x')
            ->getJson('/api/me')
            ->assertStatus(401);
    }
}
