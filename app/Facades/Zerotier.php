<?php

namespace App\Facades;

use Illuminate\Support\Facades\Facade;

/**
 * class Zerotier
 *
 * @package App\Facades
 */
class Zerotier extends Facade
{
    protected static function getFacadeAccessor(): string
    {
        return 'zerotier';
    }
}
