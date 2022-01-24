using System;
using System.Diagnostics;
using System.Runtime.CompilerServices;
using System.Runtime.InteropServices;
using System.Text;

// Make InsecureTestingSetExtraKey visible to tests
[assembly: InternalsVisibleTo("EduVpnCommonTests")]

namespace EduVpnCommon
{
	public static class Discovery
	{
		/// <summary>
		/// Verifies the signature on the JSON server_list.json/organization_list.json file.
		/// If the function returns, the signature is valid for the given file type.
		/// </summary>
		/// <param name="signatureFileContent">.minisig signature file contents.</param>
		/// <param name="signedJson">Signed .json file contents.</param>
		/// <param name="expectedFileName">The file type to be verified, one of <c>"server_list.json"</c> or <c>"organization_list.json"</c>.</param>
		/// <param name="minSignTime">Minimum time for signature. Should be set to at least the time of the previous signature.</param>
		/// <exception cref="ArgumentException">If <c>expectedFileName</c> is not one of the allowed values.</exception>
		/// <exception cref="VerifyException">If signature verification fails.</exception>
		public static void Verify(
			ArraySegment<byte> signatureFileContent,
			ArraySegment<byte> signedJson,
			string             expectedFileName,
			DateTimeOffset     minSignTime)
		{
			VerifyReturnCode result;
			{
				using var signatureHandle    = GoSliceHandle.FromArray(signatureFileContent);
				using var jsonHandle         = GoSliceHandle.FromArray(signedJson);
				using var expectedFileHandle = GoSliceHandle.FromString(expectedFileName);

				result = Verify(signatureHandle.Slice, jsonHandle.Slice, expectedFileHandle.Slice,
					(ulong) minSignTime.ToUnixTimeSeconds());
			}

			switch (result)
			{
				case VerifyReturnCode.Ok:
					return;
				case VerifyReturnCode.ErrUnknownExpectedFileName:
					throw new ArgumentException("unknown expected file name", nameof(expectedFileName));
				case VerifyReturnCode.ErrInvalidSignature:
					throw new InvalidSignatureException();
				case VerifyReturnCode.ErrInvalidSignatureUnknownKey:
					throw new InvalidSignatureUnknownKeyException();
				case VerifyReturnCode.ErrTooOld:
					throw new SignatureTooOldException();
				default:
					throw new UnknownVerifyException((sbyte) result);
			}
		}

		/// <summary>Use for testing only, see Go documentation.</summary>
		internal static void InsecureTestingSetExtraKey(string keyString)
		{
			using var keyHandle = GoSliceHandle.FromString(keyString);
			InsecureTestingSetExtraKey(keyHandle.Slice);
		}

		const string LibName = "eduvpn_common";

		[DllImport(LibName)]
		static extern VerifyReturnCode Verify(GoSlice signatureFileContent, GoSlice signedJson, GoSlice expectedFileName, ulong minSignTime);

		[DllImport(LibName)] static extern void InsecureTestingSetExtraKey(GoSlice keyStr);

		/// <summary>
		/// Safe auto-disposing Go slice handle.
		/// Non-copying alternative to `Marshal.AllocHGlobal` etc.
		/// </summary>
		class GoSliceHandle : IDisposable
		{
			GCHandle         gcHandle_;
			readonly GoSlice slice_;

			public GoSlice Slice => gcHandle_.IsAllocated
				? slice_
				: throw new InvalidOperationException("Handle was disposed");

			GoSliceHandle(Array array, int offset, int count)
			{
				Debug.Assert(offset <= array.Length && /*prevent overflow:*/ count <= array.Length && offset <= array.Length - count);
				gcHandle_ = GCHandle.Alloc(array, GCHandleType.Pinned);
				var elemSize = Marshal.SizeOf(array.GetType().GetElementType()!);
				slice_ = new GoSlice(gcHandle_.AddrOfPinnedObject() + offset * elemSize, count * elemSize);
			}

			public static GoSliceHandle FromArray<T>(ArraySegment<T> segment) where T : struct =>
				new GoSliceHandle(segment.Array!, segment.Offset, segment.Count);

			/// <summary>From string as UTF-8.</summary>
			public static GoSliceHandle FromString(string str) =>
				FromArray(new ArraySegment<byte>(Encoding.UTF8.GetBytes(str)));

			public void Dispose() => gcHandle_.Free();
		}

		// C-compatible structure
		readonly struct GoSlice
		{
			readonly IntPtr data_;
			readonly long   len_, cap_;

			public GoSlice(IntPtr data, long len, long cap)
			{
				data_ = data;
				len_  = len;
				cap_  = cap;
			}

			public GoSlice(IntPtr data, long len) : this(data, len, len) { }
		}
	}

	/// <summary>Verification failed, do not trust the file.</summary>
	public abstract class VerifyException : Exception
	{
		protected VerifyException(string message) : base(message) { }
	}

	/// <summary>Signature is invalid (for the expected file type).</summary>
	public sealed class InvalidSignatureException : VerifyException
	{
		public InvalidSignatureException() : base("invalid signature") { }
	}

	/// <summary>Signature was created with an unknown key and has not been verified.</summary>
	public sealed class InvalidSignatureUnknownKeyException : VerifyException
	{
		public InvalidSignatureUnknownKeyException() : base("invalid signature (unknown key)") { }
	}

	/// <summary>Signature timestamp smaller than specified minimum signing time (rollback).</summary>
	public sealed class SignatureTooOldException : VerifyException
	{
		public SignatureTooOldException() : base("replay of previous signature (rollback)") { }
	}

	/// <summary>Other unknown error.</summary>
	public sealed class UnknownVerifyException : VerifyException
	{
		public UnknownVerifyException(sbyte code) : base($"unknown verify error ({code})") => Debug.Assert(code != 0);
	}

	enum VerifyReturnCode : sbyte
	{
		Ok,
		ErrUnknownExpectedFileName,
		ErrInvalidSignature,
		ErrInvalidSignatureUnknownKey,
		ErrTooOld
	}
}
