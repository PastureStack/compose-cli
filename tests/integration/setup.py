from setuptools import setup

setup(
    name='RancherCliIntegrationTests',
    version='0.1',
    packages=[
      'platformtest',
      'platformtest.core',
    ],
    license='ASL 2.0',
    long_description=open('README.txt').read(),
)
