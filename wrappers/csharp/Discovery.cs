using System;
using System.Runtime.CompilerServices;
using System.Runtime.InteropServices;
using System.Text;

[assembly: InternalsVisibleTo("EduVpnCommonTests")]

namespace EduVpnCommon
{
	public static class Discovery
	{
		/// <summary>
		/// Verifies the signature on the JSON server_list.json/organization_list.json file.
		/// If the function returns the signature is valid for the given file type.
		/// </summary>
		/// <param name="signatureFileContent">.minisig signature file contents.</param>
		/// <param name="signedJson">Signed .json file contents.</param>
		/// <param name="expectedFileName">The file type to be verified, one of <c>"server_list.json"</c> or <c>"organization_list.json"</c>.</param>
		/// <param name="minSignTime">Minimum time for signature. Should be set to at least the time in a previously retrieved file.</param>
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

			if (result != VerifyReturnCode.Ok)
			{
				if (result == VerifyReturnCode.ErrUnknownExpectedFileName)
					throw new ArgumentException("unknown name", nameof(expectedFileName));
				throw new VerifyException((VerifyErrorCode) result);
			}
		}

		/// <summary>Use for testing only, see Go documentation.</summary>
		internal static void InsecureTestingSetExtraKey(string keyString)
		{
			using var keyHandle = GoSliceHandle.FromString(keyString);
			InsecureTestingSetExtraKey(keyHandle.Slice);
		}

		const string VerifyLibName = "eduvpn_verify";

		[DllImport(VerifyLibName)]
		static extern VerifyReturnCode Verify(GoSlice signatureFileContent, GoSlice signedJson, GoSlice expectedFileName, ulong minSignTime);

		[DllImport(VerifyLibName)] static extern void InsecureTestingSetExtraKey(GoSlice keyStr);

		class GoSliceHandle : IDisposable
		{
			GCHandle         gcHandle_;
			readonly GoSlice slice_;

			public GoSlice Slice => gcHandle_.IsAllocated
				? slice_
				: throw new InvalidOperationException("Handle was disposed");

			GoSliceHandle(Array array, int offset, int count)
			{
				gcHandle_ = GCHandle.Alloc(array, GCHandleType.Pinned);
				var elemSize = Marshal.SizeOf(array.GetType().GetElementType()!);
				slice_ = new GoSlice(gcHandle_.AddrOfPinnedObject() + offset * elemSize, count * elemSize);
			}

			public static GoSliceHandle FromArray<T>(ArraySegment<T> segment) where T : struct =>
				new GoSliceHandle(segment.Array!, segment.Offset, segment.Count);

			public static GoSliceHandle FromString(string str) =>
				FromArray(new ArraySegment<byte>(Encoding.UTF8.GetBytes(str)));

			public void Dispose() => gcHandle_.Free();
		}

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

	public class VerifyException : Exception
	{
		public VerifyErrorCode Code { get; }

		internal VerifyException(VerifyErrorCode code) : base(GetMessage(code)) => Code = code;

		static string GetMessage(VerifyErrorCode code) => code switch
		{
			VerifyErrorCode.ErrInvalidSignature           => "invalid signature",
			VerifyErrorCode.ErrInvalidSignatureUnknownKey => "invalid signature (unknown key)",
			VerifyErrorCode.ErrTooOld                     => "replay of previous signature (rollback)",
			_                                             => $"unknown verify error ({code})"
		};
	}

	public enum VerifyErrorCode
	{
		/// <summary>Signature is invalid (for the expected file type).</summary>
		ErrInvalidSignature = VerifyReturnCode.ErrUnknownExpectedFileName + 1,

		/// <summary>Signature was created with an unknown key and has not been verified.</summary>
		ErrInvalidSignatureUnknownKey,

		/// <summary>Signature has a timestamp lower than the specified minimum signing time.</summary>
		ErrTooOld
	}

	enum VerifyReturnCode
	{
		Ok,
		ErrUnknownExpectedFileName

		//...
	}
}