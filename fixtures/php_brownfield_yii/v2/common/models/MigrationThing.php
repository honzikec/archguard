<?php

namespace V2\Common\models;

use V2\Frontend\models\Mirror;

class MigrationThing
{
    public function mirrorClass(): string
    {
        return Mirror::class;
    }
}
