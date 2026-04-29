<?php

namespace Backend\controllers;

use Frontend\services\PromoService;
use yii\web\Controller;

class SiteController extends Controller
{
    public function actionIndex(): string
    {
        return PromoService::class;
    }
}
