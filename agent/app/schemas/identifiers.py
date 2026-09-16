from typing import Annotated

from pydantic import Field, StringConstraints


BusinessId = Annotated[int, Field(strict=True, gt=0)]
TransportId = Annotated[
    str,
    StringConstraints(strict=True, strip_whitespace=True, min_length=1),
]
