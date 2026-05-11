<?php

namespace App\Http\Controllers;

/**
 * class ViewController 展示默认视图，并注入需要的数据
 *
 * @package App\Http\Controllers
 */
class ViewController
{
    public function __invoke()
    {
        $jwt = null;
        return view('index', compact('jwt'));
    }
}
