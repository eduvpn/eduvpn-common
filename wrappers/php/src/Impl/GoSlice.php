<?php declare(strict_types=1);

/** @internal */

namespace EduVpn\Common\Impl;

use FFI;
use FFI\CData;
use RuntimeException;

/** @internal */
class GoSlice
{
	// Will be destroyed along with this GoSlice
	private CData $cData, $slice;

	public function __construct(FFI $ffi, string $data)
	{
		$len   = strlen($data);
		$cData = FFI::new(FFI::arrayType(FFI::type('char'), [$len]), false);
		if ($cData === null) throw new RuntimeException('error allocating buffer');
		$this->cData = $cData;
		FFI::memcpy($cData, $data, $len);

		$slice = $ffi->new('GoSlice');
		if ($slice === null) throw new RuntimeException('error allocating buffer');
		$this->slice = $slice;
		$slice->data = FFI::addr($cData); // $cData must not be destroyed while $slice is in use
		$slice->cap  = $slice->len = $len;
	}

	public function slice(): CData
	{
		return $this->slice;
	}

	public function __destruct()
	{
		// Make sure we do not unknowingly use a slice with deallocated data
		$this->slice->data = null;
		$this->slice->cap  = $this->slice->len = 0;
	}
}
