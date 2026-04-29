<?php

namespace Backend\assets\Component;

use Backend\assets\Bundle\AdminBundle;

class AdminWidget
{
    public function bundleClass(): string
    {
        return AdminBundle::class;
    }
}
