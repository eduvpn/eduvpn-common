<?php declare(strict_types=1);

namespace EduVpn\Common;

/** Other unknown error. */
final class UnknownVerifyException extends VerifyException
{
	public function __construct(int $code)
	{
		parent::__construct("unknown verify error ($code)", $code);
	}
}
