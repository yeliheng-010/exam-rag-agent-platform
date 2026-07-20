import base64
import io
import shutil
import unittest
from unittest import mock

from PIL import Image, ImageChops

from docreader.parser.docx_parser import _open_related_part_image
from docreader.parser.office_media import (
    attach_vector_data_uris_to_markdown,
    rasterize_media_batch,
    rasterize_media_bytes,
)


FORMULA_WMF_BASE64 = (
    "183GmgAAAAAAAAADgAICCQAAAACTXwEACQAAA+ABAAACAKcAAAAAAAUAAAACAQEAAAAFAAAAAQL///8ABQAAAC4BGQAAAAUAAAALAgAAAAAFAAAADAKAAgADCwAAACYGDwAMAE1hdGhUeXBlAABwABIAAAAmBg8AGgD/////AAAQAAAAwP///6j////AAgAAKAIAAAUAAAAJAgAAAAIFAAAAFAK8ARwAHAAAAPsCBf7jAAAAAACQAQAAAAEAAgAQU3ltYm9sAADA+D10zFUKTwAACgAAAAAAAr0Cd0AAAAAEAAAALQEAAAkAAAAyCgAAAAABAAAAewAAAAUAAAAUArwBIAIcAAAA+wIF/uMAAAAAAJABAAAAAQACABBTeW1ib2wAAMD4PXTZVAo0AAAKAAAAAAACvQJ3QAAAAAQAAAAtAQEABAAAAPABAAAJAAAAMgoAAAAAAQAAAH0AAAAFAAAAFAIDApEBHAAAAPsCIP8AAAAAAACQAQEAAAAAAgAQVGltZXMgTmV3IFJvbWFuAAAACgAAAAAAAr0Cd0AAAAAEAAAALQEAAAQAAADwAQEACQAAADIKAAAAAAEAAABuAMABBQAAABQCoAHSABwAAAD7AoD+AAAAAAAAkAEBAAAAAAIAEFRpbWVzIE5ldyBSb21hbgAAAAoAAAAAAAK9AndAAAAABAAAAC0BAQAEAAAA8AEAAAkAAAAyCgAAAAABAAAAYQAAA6cAAAAmBg8AQwFBcHBzTUZDQwEAHAEAABwBAABEZXNpZ24gU2NpZW5jZSwgSW5jLgAFAQAHBERTTVQ3AAETV2luQWxsQmFzaWNDb2RlUGFnZXMAEQVUaW1lcyBOZXcgUm9tYW4AEQNTeW1ib2wAEQVDb3VyaWVyIE5ldwARBE1UIEV4dHJhABNXaW5BbGxDb2RlUGFnZXMAEQbLzszlABIACCEvJ/JfIY8hL0dfQVDyHx5BUPQVD0EA9EX0JfSPQl9BAPQQD0NfQQDyHyCl8gol9I8h9BAPQQD0D0j0F/SPQQDyGl9EX0X0X0X0X0EPDAEAAQABAgICAgACAAEBAQADAAEABAAFAAoBABAAAAAAAAAADwEDAAIDAA8AAQAPAQIAg2EADwADABsAAAsBAA8BAgCDbgAADwABAQAACg8BAgCWewACAJZ9AAAAAAAKAAAAJgYPAAoA/////wEAAAAAABwAAAD7AhAABwAAAAAAvAIAAACGAQICIlN5c3RlbQDTSQCKAAAACgBGVWbTSQCKAAAAAAD4z9MABAAAAC0BAAAEAAAA8AEBAAMAAAAAAA=="
)


class TestOfficeMedia(unittest.TestCase):
    def test_batch_uses_imagemagick_when_office_conversion_fails(self):
        fake_png = b"\x89PNG\r\n\x1a\nconverted"
        with mock.patch(
            "docreader.parser.office_media._run_soffice_batch",
            return_value=[None],
        ), mock.patch(
            "docreader.parser.office_media._rasterize_with_imagemagick",
            return_value=fake_png,
        ) as imagemagick:
            result = rasterize_media_batch([("formula.x-wmf", b"vector")])

        self.assertEqual(result, [fake_png])
        imagemagick.assert_called_once_with(b"vector", ".wmf")

    def test_docx_fallback_opens_vector_related_part(self):
        class RelatedPart:
            partname = "/word/media/formula.x-wmf"
            blob = base64.b64decode(FORMULA_WMF_BASE64)

        output = io.BytesIO()
        Image.new("RGB", (80, 40), "white").save(output, format="PNG")
        with mock.patch(
            "docreader.parser.docx_parser.rasterize_media_bytes",
            return_value=output.getvalue(),
        ):
            image = _open_related_part_image(RelatedPart())

        self.assertIsNotNone(image)
        self.assertEqual(image.size, (80, 40))

    def test_vector_data_uri_is_replaced_with_png_reference(self):
        wmf = base64.b64decode(FORMULA_WMF_BASE64)
        markdown = f"已知 ![](data:image/x-wmf;base64,{base64.b64encode(wmf).decode()}) 成立"
        fake_png = b"\x89PNG\r\n\x1a\nfixture"

        with mock.patch(
            "docreader.parser.office_media.rasterize_media_batch",
            return_value=[fake_png],
        ):
            rendered, images = attach_vector_data_uris_to_markdown(markdown)

        self.assertNotIn("data:image/x-wmf", rendered)
        self.assertRegex(rendered, r"!\[\]\(images/[0-9a-f-]+\.png\)")
        self.assertEqual(len(images), 1)
        self.assertEqual(base64.b64decode(next(iter(images.values()))), fake_png)

    @unittest.skipUnless(shutil.which("soffice"), "LibreOffice not available")
    def test_formula_wmf_becomes_non_blank_cropped_png(self):
        png = rasterize_media_bytes(
            "formula.x-wmf",
            base64.b64decode(FORMULA_WMF_BASE64),
        )

        self.assertIsNotNone(png)
        image = Image.open(io.BytesIO(png)).convert("RGB")
        self.assertLess(image.width, 500)
        self.assertLess(image.height, 500)
        white = Image.new("RGB", image.size, "white")
        self.assertIsNotNone(ImageChops.difference(image, white).getbbox())


if __name__ == "__main__":
    unittest.main()
