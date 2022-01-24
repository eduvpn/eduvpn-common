<?php declare(strict_types=1);

namespace EduVpn\Common;

/** Signature timestamp smaller than specified minimum signing time (rollback). */
final class SignatureTooOldException extends VerifyException
{
	public function __construct()
	{
		parent::__construct('replay of previous signature (rollback)', 4);
	}
}
