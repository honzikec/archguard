<?php

namespace Backend\models;

use Common\models\Catalog;

class Order
{
    public function catalogClass(): string
    {
        return Catalog::class;
    }
}
