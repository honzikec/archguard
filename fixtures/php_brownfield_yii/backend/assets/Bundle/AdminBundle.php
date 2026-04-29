<?php

namespace Backend\assets\Bundle;

use Backend\assets\Component\AdminWidget;

class AdminBundle
{
    public function widgetClass(): string
    {
        return AdminWidget::class;
    }
}
