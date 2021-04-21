import setuptools

setuptools.setup(
    name='zqd',
    install_requires=[
        'durationpy',
        'python-dateutil',
        'requests',
    ],
    py_modules=['zqd'],
    python_requires='>=3.3',
)
