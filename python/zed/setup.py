import setuptools

setuptools.setup(
    name='zed',
    install_requires=[
        'durationpy',
        'python-dateutil',
        'requests',
    ],
    py_modules=['zed'],
    python_requires='>=3.3',
)
