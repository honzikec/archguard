<?php

namespace Common\models;

use Frontend\forms\SignupForm;

class User
{
    public function formClass(): string
    {
        return SignupForm::class;
    }
}
