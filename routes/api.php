<?php

/** @var RouteFileRegistrar $router */

use App\Http\Controllers\ZerotierController;
use Illuminate\Routing\RouteFileRegistrar;
use Illuminate\Routing\Router;

$router->group([
], function (Router $router) {
    $router->get('captcha', function () {})->name('captcha');
});

$router->group([
    'middleware' => 'jwt',
], function (Router $router) {
    $router->get('/status', [ZerotierController::class, 'status']);
    $router->get('/networks', [ZerotierController::class, 'networks']);
    $router->post('/networks', [ZerotierController::class, 'createNetwork']);
    $router->get('/networks/{nwid}', [ZerotierController::class, 'network']);
    $router->get('/networks/{nwid}/members', [ZerotierController::class, 'members']);
    $router->post('/networks/{nwid}/members/{id}', [ZerotierController::class, 'updateMember']);
    $router->delete('/networks/{nwid}/members/{id}', [ZerotierController::class, 'deleteMember']);
});
