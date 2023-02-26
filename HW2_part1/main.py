import base64
from typing import Union
from fastapi import FastAPI, HTTPException, UploadFile, Form, File
from fastapi.encoders import jsonable_encoder
from pydantic import BaseModel, ValidationError
from starlette import status


class Product(BaseModel):
    id: Union[int, None]
    name: str
    description: Union[str, None]
    icon: Union[bytes, None]


app = FastAPI()
allProducts: [int, Product] = {}
emptyId = 0


@app.get("/products")
def read_root():
    return list(allProducts.values())


@app.get("/products/{product_id}")
def read_product(product_id: int):
    if product_id not in allProducts:
        raise HTTPException(status_code=404, detail="Product not found")
    return allProducts[product_id]


@app.post("/products")
async def add_product(product: Product):
    global emptyId
    product.id = emptyId
    allProducts[emptyId] = product
    emptyId += 1
    return product


@app.put("/products/{product_id}")
async def update_product(product_id: int, product: Product):
    product.id = product_id
    allProducts[product_id] = product
    return product


@app.delete("/products/{product_id}")
def delete_product(product_id: int):
    if product_id not in allProducts:
        raise HTTPException(status_code=404, detail="Product not found")
    del allProducts[product_id]
    return {"ok": True}


@app.put("/products/icons/{product_id}")
async def update_product(product_id: int, icon: UploadFile = File(...)):
    if product_id not in allProducts:
        raise HTTPException(status_code=404, detail="Product not found")
    data = await icon.read()
    allProducts[product_id].icon = base64.b64encode(data).decode()
    return allProducts[product_id]

"""
@app.post("/products")
async def add_product(product: str = Form(...), icon: Union[UploadFile, None] = None):
    try:
        product = Product.parse_raw(product)
    except ValidationError as e:
        raise HTTPException(
            detail=jsonable_encoder(e.errors()),
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY
        ) from e
    global emptyId
    product.id = emptyId
    if icon is not None:
        data = await icon.read()
        output = base64.b64encode(data).decode()
        product.icon = output
    allProducts[emptyId] = product
    emptyId += 1
    return product

@app.put("/products/{product_id}")
async def update_product(product_id: int, product: str = Form(...), icon: Union[UploadFile, None] = None):
    if product_id not in allProducts:
        raise HTTPException(status_code=404, detail="Product not found")
    try:
        product = Product.parse_raw(product)
    except ValidationError as e:
        raise HTTPException(
            detail=jsonable_encoder(e.errors()),
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY
        ) from e
    product.id = product_id
    if icon is not None:
        data = await icon.read()
        output = base64.b64encode(data).decode()
        product.icon = output
    allProducts[product_id] = product
    return product
"""