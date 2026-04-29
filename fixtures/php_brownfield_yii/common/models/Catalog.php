<?php

namespace Common\models;

use Frontend\models\Landing;

class Catalog
{
    public function nextStep(): string
    {
        return Landing::class;
    }
}
