<?php

namespace Tests\Feature;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class ZerotierApiTest extends TestCase
{
    use RefreshDatabase;

    protected string $token = '';

    protected function setUp(): void
    {
        parent::setUp();
        config()->set('jwt.secret', 'unit-test-secret');
        config()->set('zerotier.base_url', 'http://127.0.0.1:9993');
        config()->set('zerotier.token', 'zt-local-token');
        config()->set('zerotier.node_address', null);

        $user      = User::factory()->create(['email' => 'admin@example.com', 'password' => 'password']);
        $this->token = app(\App\Services\JwtService::class)->issue($user->id);
    }

    public function test_status_sends_local_auth_header(): void
    {
        Http::fake([
            '127.0.0.1:9993/status' => Http::response(['address' => 'abcdef0123', 'version' => '1.12.1', 'online' => true], 200),
        ]);

        $this->withToken($this->token)->getJson('/api/status')
            ->assertOk()
            ->assertJsonPath('result.version', '1.12.1');

        Http::assertSent(fn ($req) => $req->hasHeader('X-ZT1-Auth', 'zt-local-token')
            && str_ends_with($req->url(), '/status'));
    }

    public function test_networks_list_expands_ids_to_config(): void
    {
        Http::fake([
            '127.0.0.1:9993/controller/network' => Http::response(['8056c2e21c000001'], 200),
            '127.0.0.1:9993/controller/network/*' => Http::response([
                'id' => '8056c2e21c000001', 'nwid' => '8056c2e21c000001',
                'name' => 'home', 'private' => true, 'active' => true,
            ], 200),
        ]);

        $this->withToken($this->token)->getJson('/api/networks')
            ->assertOk()
            ->assertJsonPath('result.0.nwid', '8056c2e21c000001')
            ->assertJsonPath('result.0.name', 'home')
            ->assertJsonPath('result.0.active', true);
    }

    public function test_create_network_builds_nwid_from_node_address_and_expands_cidr(): void
    {
        Http::fake([
            '127.0.0.1:9993/status' => Http::response(['address' => 'abcdef0123'], 200),
            '127.0.0.1:9993/controller/network/*' => Http::response(['id' => 'x'], 200),
        ]);

        $this->withToken($this->token)->postJson('/api/networks', [
            'name'  => 'office',
            'cidr'  => '10.147.20.0/24',
            'private' => true,
        ])->assertOk()->assertJsonPath('status', 0);

        Http::assertSent(function ($req) {
            // nwid 前 10 位 = 控制器节点地址
            $okUrl = (bool) preg_match('#/controller/network/abcdef0123[0-9a-f]{6}$#', $req->url());
            $body  = json_decode($req->body(), true);

            return $okUrl
                && ($body['active'] ?? null) === true
                && ($body['name'] ?? null) === 'office'
                && ($body['routes'][0]['target'] ?? null) === '10.147.20.0/24'
                && ($body['ipAssignmentPools'][0]['ipRangeStart'] ?? null) === '10.147.20.1';
        });
    }

    public function test_update_member_strips_readonly_fields(): void
    {
        Http::fake([
            '127.0.0.1:9993/controller/network/*/member/*' => Http::response(['authorized' => true], 200),
        ]);

        $this->withToken($this->token)->postJson('/api/networks/nwid1/members/node1', [
            'authorized'      => true,
            'name'            => 'laptop',
            'lastOnline'      => 123,
            'physicalAddress' => '1.2.3.4',
        ])->assertOk()->assertJsonPath('status', 0);

        Http::assertSent(function ($req) {
            $body = json_decode($req->body(), true);

            return ($body['authorized'] ?? null) === true
                && ($body['name'] ?? null) === 'laptop'
                && !array_key_exists('lastOnline', $body)
                && !array_key_exists('physicalAddress', $body);
        });
    }

    public function test_unauthenticated_request_is_rejected_before_hitting_controller(): void
    {
        Http::fake();
        $this->getJson('/api/networks')->assertStatus(401);
        Http::assertNothingSent();
    }
}
