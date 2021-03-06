from setuptools import setup

setup(
    name="semanticizest",
    description="Python wrapper for Semanticizer go implementation",
    url="https://github.com/semanticize/st",
    version="0.0.1",
    packages=["semanticizest"],
    package_dir={"semanticizest": "python/semanticizest"},
    classifiers=[
        "Intended Audience :: Science/Research",
        "License :: OSI Approved :: Apache Software License",
        "Topic :: Scientific/Engineering",
        "Topic :: Scientific/Engineering :: Information Analysis",
        "Topic :: Text Processing",
    ],
    install_requires=[]
)
