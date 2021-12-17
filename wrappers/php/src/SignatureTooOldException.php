<?php declare(strict_types=1);

namespace EduVpn\Common;

/** Signature has a timestamp lower than the specified minimum signing time. */
final class SignatureTooOldException extends VerifyException
{
	public function __construct()
	{
		parent::__construct('replay of previous signature (rollback)', 4);
	}
}
