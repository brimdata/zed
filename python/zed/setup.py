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
    setup_requires=['setuptools_scm'],
    use_scm_version={
        'fallback_version': 'unknown',
        'root': '../..',
    },
)
