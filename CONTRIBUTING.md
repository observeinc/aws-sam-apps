# Contributing to AWS SAM Applications for Observe Inc

Thank you for your interest in contributing to the AWS SAM applications for Observe Inc! We value your contributions and want to make sure the process is easy and beneficial for everyone involved. Please follow these guidelines to contribute effectively to the project.

## Communication

Before starting work on a significant change, we encourage you to open an issue to discuss your proposed contribution. This allows for collaborative decision-making and avoids unnecessary work or duplicate efforts.

## Contribution Process

1. **Open an Issue:**
   Begin by opening a new issue in the GitHub repository. Clearly describe the bug, feature, or enhancement with as much detail as possible.

2. **Fork and Clone the Repository:**
   Fork the project on GitHub and then clone your fork locally.

3. **Create a Branch:**
   Create a branch in your local clone that follows our branch naming convention, which is based on semantic commits. For example, `feat/add-new-metric` or `fix/log-output`.

4. **Make Your Changes:**
   Make the changes you propose in your branch, following the coding standards provided below.

5. **Commit Your Changes:**
   Commit your changes using semantic commit messages. This helps us understand the purpose of your changes and can automate the release process. For example, `feat: add new metric endpoint`.

6. **Push Your Changes:**
   Push your changes to your fork on GitHub.

7. **Submit a Pull Request:**
   From your fork, submit a pull request to the main repository. In your pull request, reference the initial issue you opened and provide a concise description of the changes as well as any additional information that might be relevant.

8. **Review and Merge:**
   Maintain the maintainers will review your pull request. Be open to discussion and additional changes. Once approved, your pull request will be merged into the main branch.

## Semantic Commit Messages

We use semantic commit messages to streamline the release process and create a readable commit history. Your commit messages should follow this format:

```markdown
<type>: <subject>
```

Types include:

- `feat`: A new feature.
- `fix`: A bug fix.
- `docs`: Documentation changes.
- `style`: Formatting, missing semi-colons, etc; no code change.
- `refactor`: Refactoring production code.
- `test`: Adding tests, refactoring test; no production code change.
- `chore`: Updating build tasks, package manager configs, etc; no production code change.

## Branch Naming Convention

Your branch names should also follow the semantic format, prefixed with the type of changes:

```markdown
<type>/<short-description>
```

For example:

- `feat/add-metric-endpoint`
- `fix/data-sync-issue`
- `chore/update-dependencies`

## Testing

Add new tests for your changes and run the full test suite to ensure that existing tests pass.

## Pull Request Process

In your pull request, please:

- Remove any non-essential build or install dependencies.
- Update the documentation to reflect your changes if applicable.
- Increment the version numbers in any examples and the `README.md` to the new version that your Pull Request would represent, following [SemVer](http://semver.org/).

## Questions?

If you have questions or need further clarification, please open an issue, and we'll be happy to assist.

We're excited to welcome you to our community and see your contributions to the AWS SAM applications for Observe Inc!
