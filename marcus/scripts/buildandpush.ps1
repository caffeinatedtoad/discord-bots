# Run from /discord-bots/marcus

param (
    [String]
    $imageName,
    [String]
    $imageTag,
    [String]
    $registry,
    [Switch]
    $buildBase,
    # Only applies when building base image
    [Switch]
    $arm
)


Write-Host ""
function Print-Usage() {
    Write-Host "-------------------------------------------------------"
    Write-Host "buildandpush.ps1 - Build the marcus image and push it"
    Write-Host ""
    Write-Host "usage:  "
    Write-Host "      Build the marcus image"
    Write-Host "        ./scripts/buildandpush.ps1 -imageName 'marcus' -imageTag 'mytag' "
    Write-Host "      Build the marcus image for arm systems (requires an arm base image)"
    Write-Host "        ./scripts/buildandpush.ps1 -imageName 'marcus' -imageTag 'mytag' -arm"
    Write-Host "      Build the image and push it"
    Write-Host "        ./scripts/buildandpush.ps1 -imageName 'marcus' -imageTag 'mytag' -registry 'myRegistry'"
    Write-Host "      Build the base marcus image (-arm can also be passed here)"
    Write-Host "        ./scripts/buildandpush.ps1 -imageName 'marcus-base' -imageTag 'latest' -buildBase"
    Write-Host "-------------------------------------------------------"
}

if ($imageName -eq "") {
    Write-Host "Error: Must provide the image name"
    Print-Usage
    exit 1
}

if ($imageTag -eq "") {
    Write-Host "Error: Must provide an image tag"
    Print-Usage
    exit 1
}

$dockerFile = "package/Dockerfile"
if ($buildBase) {
    $dockerFile = $dockerFile + "-base"
}

if ($arm) {
    $dockerFile = $dockerFile + '-arm'
}

$fullImage = "$imageName`:$imageTag"

if ($registry -ne "") {
    $fullImage = "$registry/" + $fullImage
}

docker build -f $dockerFile -t $fullImage .

if ($registry -ne "" )
{
    docker push $fullImage
}
### supporting functions ###
