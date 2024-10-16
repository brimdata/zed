import setuptools

setuptools.setup(
    name='superdb',
    install_requires=[
        'durationpy',
        'python-dateutil',
        'requests',
    ],
    py_modules=['superdb'],
    python_requires='>=3.3',
    setup_requires=['setuptools_scm'],
    use_scm_version={
        'fallback_version': '0+unknown',
        'root': '../..',
        'version_scheme': 'post-release',
    },
)
