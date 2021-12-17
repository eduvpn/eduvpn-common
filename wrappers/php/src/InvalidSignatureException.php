<?php declare(strict_types=1);

namespace EduVpn\Common;

/** Signature is invalid (for the expected file type). */
final class InvalidSignatureException extends VerifyException
{
	public function __construct()
	{
		parent::__construct('invalid signature', 2);
	}
}
