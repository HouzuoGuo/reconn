import pytest
from pathlib import Path
import os
from . import create_app


@pytest.fixture()
def app():
    test_data_dir = "/tmp/reconn-voicesvc-test"
    os.mkdir(test_data_dir)
    app = create_app(test_data_dir)
    yield app
    Path(test_data_dir).rmdir()


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
