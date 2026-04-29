<?php

namespace Frontend\models;

use V2\Common\models\MigrationThing;

class Landing
{
    public function migrationClass(): string
    {
        return MigrationThing::class;
    }
}
