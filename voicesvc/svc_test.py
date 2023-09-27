import pytest
import shutil
import os
import svc
from app import create_app


@pytest.fixture()
def app():
    test_data_dir = "/tmp/reconn-voicesvc-test"
    os.makedirs(test_data_dir, exist_ok=True)
    voice_svc = svc.VoiceSvc()
    voice_svc.init_clone("cpu", test_data_dir)
    flask_app = create_app(
        test_data_dir, test_data_dir, test_data_dir, "cpu", voice_svc
    )
    yield flask_app
    # Clean-up after executing the tests.
    shutil.rmtree(test_data_dir)


@pytest.fixture()
def client(app):
    return app.test_client()


@pytest.fixture()
def runner(app):
    return app.test_cli_runner()


def test_readback(client):
    resp = client.get("/readback")
    assert (
        b'{"request-host":"localhost","request-method":"GET","request-url":"http://localhost/readback"}'
        in resp.data.strip()
    )
