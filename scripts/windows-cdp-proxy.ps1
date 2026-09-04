param(
    [Parameter(Mandatory = $true)]
    [string]$ListenAddress,
    [int]$ListenPort = 9337,
    [string]$TargetAddress = "127.0.0.1",
    [int]$TargetPort = 9336
)

$ErrorActionPreference = "Stop"

$listenIP = [System.Net.IPAddress]::Parse($ListenAddress)
$listener = [System.Net.Sockets.TcpListener]::new($listenIP, $ListenPort)
$listener.Start()

Write-Host "Temporary Chrome DevTools bridge"
Write-Host "  $ListenAddress`:$ListenPort -> $TargetAddress`:$TargetPort"
Write-Host "Keep this window open during the take. Press Ctrl+C to stop."

try {
    while ($true) {
        $client = $listener.AcceptTcpClient()
        $target = [System.Net.Sockets.TcpClient]::new()
        try {
            $target.Connect($TargetAddress, $TargetPort)
            $clientStream = $client.GetStream()
            $targetStream = $target.GetStream()
            $toTarget = $clientStream.CopyToAsync($targetStream)
            $toClient = $targetStream.CopyToAsync($clientStream)
            $copyTasks = [System.Threading.Tasks.Task[]]@($toTarget, $toClient)
            [System.Threading.Tasks.Task]::WaitAny($copyTasks) | Out-Null
        }
        catch {
            Write-Warning $_.Exception.Message
        }
        finally {
            $target.Dispose()
            $client.Dispose()
        }
    }
}
finally {
    $listener.Stop()
}
