<?php

/** @var Router $router */

use App\Http\Controllers\ViewController;
use Illuminate\Routing\Router;

$router->get('/', ViewController::class)->name('index');
$router->fallback(ViewController::class)->name('fallback');
