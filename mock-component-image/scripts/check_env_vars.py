import os
import shutil

if shutil.which("docker") is None:
    raise Exception("Docker not installed! Go install Docker!")

if shutil.which("helm") is None:
    raise Exception("Helm not installed! Go install Helm 3 CLI!")

# NW this can be set when the code is in the mch operator
_product_version = os.environ.get('PRODUCT_VERSION')
if not _product_version:
    raise Exception("You must export PRODUCT_VERSION!")

_image_remote = os.environ.get('MOCK_IMAGE_REMOTE')
if not _image_remote:
    raise Exception("You must export MOCK_IMAGE_REMOTE! (ex: 'MOCK_IMAGE_REMOTE/MOCK_IMAGE_NAME:MOCK_IMAGE_TAG')")

_image_name = os.environ.get('MOCK_IMAGE_NAME')
if not _image_name:
    raise Exception("You must export MOCK_IMAGE_NAME! (ex: 'MOCK_IMAGE_REMOTE/MOCK_IMAGE_NAME:MOCK_IMAGE_TAG')")

_image_tag = os.environ.get('MOCK_IMAGE_TAG')
if not _image_tag:
    raise Exception("You must export MOCK_IMAGE_TAG! (ex: 'MOCK_IMAGE_REMOTE/MOCK_IMAGE_NAME:MOCK_IMAGE_TAG')")
