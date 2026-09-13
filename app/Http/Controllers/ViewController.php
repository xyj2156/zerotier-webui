<?php

namespace App\Http\Controllers;

/**
 * class ViewController 展示默认视图，并注入需要的数据
 *
 * @package App\Http\Controllers
 */
class ViewController extends Controller
{
    public function __invoke()
    {
        return view('index');
    }
}
