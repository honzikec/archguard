<?php

namespace V2\Frontend\models;

use Backend\models\Order;

class Mirror
{
    public function orderClass(): string
    {
        return Order::class;
    }
}
