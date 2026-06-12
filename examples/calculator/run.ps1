# MOFU Calculator Launcher
# Sets up proper terminal for ANSI rendering

$Host.UI.RawUI.WindowTitle = "MOFU Calculator"

# Enable VT processing
$code = @'
using System;
using System.Runtime.InteropServices;
public class VT {
    [DllImport("kernel32.dll", SetLastError = true)]
    static extern bool GetConsoleMode(IntPtr hConsole, out uint mode);
    [DllImport("kernel32.dll", SetLastError = true)]
    static extern bool SetConsoleMode(IntPtr hConsole, uint mode);
    [DllImport("kernel32.dll")]
    static extern IntPtr GetStdHandle(int nStdHandle);

    public static void Enable() {
        IntPtr handle = GetStdHandle(-11); // STD_OUTPUT_HANDLE
        uint mode;
        GetConsoleMode(handle, out mode);
        mode |= 0x0004; // ENABLE_VIRTUAL_TERMINAL_PROCESSING
        SetConsoleMode(handle, mode);
    }
}
'@
Add-Type -TypeDefinition $code
[VT]::Enable()

cd C:\Users\Ben\workspace\mofu\examples\calculator
go run main.go
