<?php
use App\Infra\Db;

function load_user() {
    return Db::connect();
}
