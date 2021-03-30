from setuptools import setup

setup(
    name = 'brim',
    version = '0.0.1',
    description = 'Brim tools for ZNG data',
    url = 'https://github.com/brimdata/zed',
    author = 'Brim Security, Inc.',
    author_email = 'support@brimsecurity.com',
    license = 'BSD 3-Clause License',
    package_dir={"": "src"},
    packages=["brim"],
    ext_package="brim",
    setup_requires=['wheel','cffi>=1.12.0'],
    install_requires=['cffi>=1.12.0'],
    cffi_modules=["src/build_zqext.py:ffibuilder"],
)
