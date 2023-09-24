import pytest
from . import create_app


@pytest.fixture()
def app():
    app = create_app()
    yield app


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
