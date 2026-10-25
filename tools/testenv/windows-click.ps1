# usage: windows-click.ps1 X Y [left|right|move|double|drag X2 Y2]
# Moves the pointer on the interactive desktop and presses a button, as the Windows
# test VM has no xdotool. "drag" holds the left button from X,Y to X2,Y2. Upload it
# with upload.sh and run it through exec.sh cmd.
param([int]$X, [int]$Y, [string]$Button = "left", [int]$X2 = 0, [int]$Y2 = 0)
Add-Type -Namespace W -Name U -MemberDefinition @'
[System.Runtime.InteropServices.DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
[System.Runtime.InteropServices.DllImport("user32.dll")] public static extern void mouse_event(uint f, int dx, int dy, uint d, System.UIntPtr e);
'@
function Press { [W.U]::mouse_event(2,0,0,0,[UIntPtr]::Zero); Start-Sleep -Milliseconds 60; [W.U]::mouse_event(4,0,0,0,[UIntPtr]::Zero) }
[W.U]::SetCursorPos($X, $Y) | Out-Null
Start-Sleep -Milliseconds 250
switch ($Button) {
  "left"   { Press }
  "double" { Press; Start-Sleep -Milliseconds 80; Press }
  "right"  { [W.U]::mouse_event(8,0,0,0,[UIntPtr]::Zero); Start-Sleep -Milliseconds 60; [W.U]::mouse_event(16,0,0,0,[UIntPtr]::Zero) }
  "drag"   {
    [W.U]::mouse_event(2,0,0,0,[UIntPtr]::Zero)
    for ($i = 1; $i -le 20; $i++) {
      [W.U]::SetCursorPos([int]($X + ($X2 - $X) * $i / 20), [int]($Y + ($Y2 - $Y) * $i / 20)) | Out-Null
      Start-Sleep -Milliseconds 30
    }
    Start-Sleep -Milliseconds 200
    [W.U]::mouse_event(4,0,0,0,[UIntPtr]::Zero)
  }
}
