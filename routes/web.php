<?php

/** @var Router $router */

use App\Http\Controllers\ViewController;
use Illuminate\Routing\Router;

$router->get('/', ViewController::class)->name('index')->tier('public')->forgeAlias('home.index');
$router->fallback(ViewController::class)->name('fallback')->tier('public')->forgeAlias('home.fallback');
