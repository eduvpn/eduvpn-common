<?php declare(strict_types=1);

namespace EduVpn\Common;

/** Signature was created with an unknown key and has not been verified. */
final class InvalidSignatureUnknownKeyException extends VerifyException
{
	public function __construct()
	{
		parent::__construct('invalid signature (unknown key)', 3);
	}
}
