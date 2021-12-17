<?php declare(strict_types=1);

namespace EduVpn\Common;

use Exception;

/** Verification failed, do not trust the file. */
abstract class VerifyException extends Exception
{
	public function __construct(string $message, int $code)
	{
		parent::__construct($message, $code);
	}
}
