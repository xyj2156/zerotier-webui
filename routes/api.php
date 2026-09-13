<?php

/** @var RouteFileRegistrar $router */

use App\Http\Controllers\LoginController;
use App\Http\Controllers\ZerotierController;
use Illuminate\Routing\RouteFileRegistrar;
use Illuminate\Routing\Router;

$router->group([
    'tier' => 'public',
], function (Router $router) {
    $router->post('/login', [LoginController::class, 'login'])->name('login')->forgeAlias('auth.login');
});

$router->group([
    'middleware' => 'jwt',
    'tier'       => 'admin',
], function (Router $router) {
    // 账号
    $router->get('/me', [LoginController::class, 'me'])->name('me');
    $router->post('/logout', [LoginController::class, 'logout'])->name('logout');

    // 状态
    $router->get('/status', [ZerotierController::class, 'status'])->name('status');

    // 网络
    $router->get('/networks', [ZerotierController::class, 'networks'])->name('networks');
    $router->get('/network-count', [ZerotierController::class, 'networkCount'])->name('network-count');
    $router->post('/networks', [ZerotierController::class, 'createNetwork'])->name('networks.store');
    $router->get('/networks/{nwid}', [ZerotierController::class, 'network'])->name('networks.show');
    $router->post('/networks/{nwid}', [ZerotierController::class, 'updateNetwork'])->name('networks.update');
    $router->delete('/networks/{nwid}', [ZerotierController::class, 'deleteNetwork'])->name('networks.destroy');

    // 成员
    $router->get('/networks/{nwid}/members', [ZerotierController::class, 'members'])->name('networks.members');
    $router->get('/networks/{nwid}/members/{id}', [ZerotierController::class, 'member'])->name('networks.members.show');
    $router->post('/networks/{nwid}/members/{id}', [ZerotierController::class, 'updateMember'])->name('networks.members.update');
    $router->delete('/networks/{nwid}/members/{id}', [ZerotierController::class, 'deleteMember'])->name('networks.members.destroy');

    // 通用只读透传（访问未封装的本地控制器端点）
    $router->get('/raw', [ZerotierController::class, 'raw'])->name('raw');
});
