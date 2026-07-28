# T10 (uiux): gera og-image.png 1200x630 para SEO/social cards.
# Background canvas #0a0a0f, "RESMA" em accent green, tagline em ink,
# features em body, border brand. Sem dependencias alem de System.Drawing.

Add-Type -AssemblyName System.Drawing

$out = 'D:\allt\resma\docs-site\static\img\og-image.png'

$bmp = New-Object System.Drawing.Bitmap(1200, 630)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAliasGridFit

$bg = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 10, 10, 15))
$g.FillRectangle($bg, 0, 0, 1200, 630)

$accent = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 62, 207, 142))
$brand = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 37, 99, 235))
$ink = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 244, 244, 246))
$body = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 205, 205, 205))

$titleFont = New-Object System.Drawing.Font('Arial', 96, [System.Drawing.FontStyle]::Bold)
$tagFont = New-Object System.Drawing.Font('Arial', 36, [System.Drawing.FontStyle]::Regular)
$featFont = New-Object System.Drawing.Font('Arial', 28, [System.Drawing.FontStyle]::Regular)

$g.DrawString('RESMA', $titleFont, $accent, 100, 180)
$g.DrawString('RESource MAnager for Docker Swarm', $tagFont, $ink, 100, 320)
$g.DrawString('Metrics - ML recommendations - Leak detection', $featFont, $body, 100, 400)

$pen = New-Object System.Drawing.Pen($brand, 4)
$g.DrawRectangle($pen, 40, 40, 1120, 550)

$bmp.Save($out, [System.Drawing.Imaging.ImageFormat]::Png)

$g.Dispose()
$bmp.Dispose()

Write-Output ('OG image saved: ' + $out)
Get-Item $out | Select-Object Name, Length
