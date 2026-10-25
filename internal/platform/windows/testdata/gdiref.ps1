# Renders strings the way a Win32 or WinForms application draws a label: GDI text through
# TextRenderer on a 24-bit bitmap, which the system draws with ClearType (or the smoothing the
# user chose), for the fidelity test.
#
#   gdiref.ps1 cases.json outdir
param([string]$Cases, [string]$Out)
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$list = Get-Content -Raw -Encoding UTF8 $Cases | ConvertFrom-Json
foreach ($case in $list) {
    $style = [System.Drawing.FontStyle]::Regular
    if ($case.weight -eq '700') { $style = [System.Drawing.FontStyle]::Bold }
    $font = New-Object System.Drawing.Font('Segoe UI', [single]$case.size, $style, [System.Drawing.GraphicsUnit]::Pixel)
    $bitmap = New-Object System.Drawing.Bitmap ([int]$case.width), ([int]$case.height), ([System.Drawing.Imaging.PixelFormat]::Format24bppRgb)
    $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
    $back = [System.Drawing.ColorTranslator]::FromHtml($case.bg)
    $front = [System.Drawing.ColorTranslator]::FromHtml($case.fg)
    $graphics.Clear($back)
    $flags = [System.Windows.Forms.TextFormatFlags]::NoPadding -bor [System.Windows.Forms.TextFormatFlags]::NoPrefix -bor [System.Windows.Forms.TextFormatFlags]::Left -bor [System.Windows.Forms.TextFormatFlags]::Top -bor [System.Windows.Forms.TextFormatFlags]::SingleLine
    [System.Windows.Forms.TextRenderer]::DrawText($graphics, $case.text, $font, (New-Object System.Drawing.Point 0, 0), $front, $back, $flags)
    $graphics.Dispose()
    $bitmap.Save((Join-Path $Out ($case.name + '.png')), [System.Drawing.Imaging.ImageFormat]::Png)
    $bitmap.Dispose()
    $font.Dispose()
}
