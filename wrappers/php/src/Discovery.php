<?php declare(strict_types=1);

namespace EduVpn\Common;

use EduVpn\Common\Impl\GoSlice;
use Error;
use FFI;
use InvalidArgumentException;

final class Discovery
{
	public function __construct() { }

	private static ?FFI $ffi = null;

	private static function ffi(): FFI
	{
		if (!self::$ffi) {
			if (!(self::$ffi = FFI::load(__DIR__ . '/headers/eduvpn_verify_php.h')))
				throw new Error('failed to load eduvpn_verify');
		}
		return self::$ffi;
	}

	/**
	 * Verifies the signature on the JSON server_list.json/organization_list.json file.
	 * If the function returns, the signature is valid for the given file type.
	 *
	 * @param string $signature        .minisig signature file contents.
	 * @param string $signedJson       Signed .json file contents.
	 * @param string $expectedFileName The file type to be verified, one of "server_list.json" or
	 *                                 "organization_list.json".
	 * @param int    $minSignTime      Minimum time for signature. Should be set to at least the time in a previously
	 *                                 retrieved file.
	 * @return void
	 * @throws InvalidArgumentException If expectedFileName is not one of the allowed values.
	 * @throws VerifyException If signature verification fails.
	 */
	public static function verify(string $signature, string $signedJson, string $expectedFileName,
		  int $minSignTime): void
	{
		$ffi              = self::ffi();
		$signatureData    = new GoSlice($ffi, $signature);
		$jsonData         = new GoSlice($ffi, $signedJson);
		$expectedNameData = new GoSlice($ffi, $expectedFileName);

		$result = $ffi->Verify($signatureData->slice(), $jsonData->slice(), $expectedNameData->slice(), $minSignTime);

		switch ($result) {
			case 0:
				return;
			case 1:
				throw new InvalidArgumentException('unknown expected file name', $result);
			case 2:
				throw new InvalidSignatureException();
			case 3:
				throw new InvalidSignatureUnknownKeyException();
			case 4:
				throw new SignatureTooOldException();
			default:
				throw new UnknownVerifyException($result);
		}
	}

	/** @internal Use for testing only, see Go documentation. */
	public static function insecureTestingSetExtraKey(string $keyString): void
	{
		$ffi     = self::ffi();
		$keyData = new GoSlice($ffi, $keyString);
		$ffi->InsecureTestingSetExtraKey($keyData->slice());
	}
}
