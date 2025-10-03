# Contributing to Colino

Thank you for your interest in contributing to Colino! We welcome contributions of all kinds—code, documentation, bug reports, feature requests, and community feedback.

## Our Philosophy

Colino is built on the principle of **intentional information consumption**. When contributing, keep these values in mind:

- **Privacy First**: Always prioritize user privacy and local control
- **Simplicity**: Make it easy for users to understand and use
- **Quality**: Focus on high-value, meaningful contributions
- **Accessibility**: Lower barriers to entry for non-technical users

## Ways to Contribute

### 🐛 Report Bugs

Found something not working as expected? Please file an issue:

1. **Search existing issues** first to avoid duplicates
2. **Use descriptive titles** that clearly identify the problem
3. **Include reproduction steps** and system information
4. **Add relevant logs** or error messages

**Bug Report Template:**
```markdown
## Bug Description
Brief description of what's happening

## Steps to Reproduce
1. Run command `...`
2. Do action `...`
3. See error `...`

## Expected Behavior
What should have happened

## System Information
- OS: [e.g., macOS 14.0]
- Colino version: [e.g., 0.2.0-alpha]
- Go version (if building from source): [e.g., 1.23.0]

## Additional Context
Any other relevant information
```

### 💡 Suggest Features

Have an idea that would make Colino better? We'd love to hear it!

1. **Check the roadmap** to see if it's already planned
2. **Consider the philosophy** - does it align with intentional information consumption?
3. **Think about simplicity** - how can this be implemented elegantly?

**Feature Request Template:**
```markdown
## Feature Description
Clear description of the proposed feature

## Problem Statement
What problem does this solve for users?

## Proposed Solution
How you envision this working

## Alternatives Considered
Other approaches you've thought about

## Additional Context
Why this matters to you or the community
```

### 📝 Improve Documentation

Help us make Colino more accessible:

- **Fix typos and grammatical errors**
- **Clarify confusing sections**
- **Add practical examples**
- **Translate to other languages** (eventually)
- **Create tutorials and guides**

### 💻 Code Contributions

We welcome code contributions that align with our values. Here's how to get started:

#### Development Setup

1. **Fork and clone** the repository
2. **Install Go 1.23+** if you haven't already
3. **Set up git hooks** for code quality:
   ```bash
   git config core.hooksPath .githooks
   chmod +x .githooks/pre-commit
   ```
4. **Build and test**:
   ```bash
   go build -o colino ./cmd/colino
   ./colino --version
   go test ./...
   ```

#### Making Changes

1. **Create a feature branch** from `main` or `daemon`
2. **Write clear, focused commits** with descriptive messages
3. **Follow the existing code style** (the git hooks will help)
4. **Add tests** for new functionality
5. **Update documentation** if needed
6. **Ensure all tests pass** before submitting

#### Areas Where We Need Help

- **Cross-platform support**: Windows, Linux packaging and integration
- **RSS feed improvements**: Better handling of various feed formats
- **Content extraction**: Enhanced article and video processing
- **User experience**: CLI improvements, better error messages
- **MCP enhancements**: New tools and capabilities for AI integration

### 🧪 Testing and Feedback

Help us improve quality by:

- **Testing on different platforms** and configurations
- **Trying edge cases** and unusual RSS feeds
- **Providing feedback** on setup and configuration
- **Reporting performance issues** with large feed collections

## Development Guidelines

### Code Style

- **Follow Go conventions** and use the provided git hooks
- **Write clear, self-documenting code**
- **Keep functions focused** and small
- **Add comments** for complex logic
- **Prefer simplicity** over cleverness

### Testing

- **Write tests** for new functionality
- **Test edge cases** and error conditions
- **Ensure backward compatibility** when possible
- **Test on multiple platforms** if you can

### Documentation

- **Update relevant docs** when changing functionality
- **Use clear, user-friendly language**
- **Include examples** for new features
- **Consider non-technical users** in your writing

## Submitting Changes

1. **Fork the repository** and create your feature branch
2. **Make your changes** following the guidelines above
3. **Test thoroughly** on your system
4. **Update documentation** as needed
5. **Commit your changes** with clear, descriptive messages
6. **Push to your fork** and create a pull request

### Pull Request Template

```markdown
## Description
Brief description of what this PR does

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation improvement
- [ ] Code quality improvement
- [ ] Other (please describe)

## Testing
- [ ] I have tested this change
- [ ] I have added necessary tests
- [ ] All tests pass on my system

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated if needed
- [ ] Breaking changes documented (if any)

## Additional Context
Any other information reviewers should know
```

## Community Guidelines

### Code of Conduct

We're committed to providing a welcoming and inclusive environment. Please:

- **Be respectful** and considerate
- **Focus on constructive feedback**
- **Welcome newcomers** and help them learn
- **Assume good intentions**

### Getting Help

- **GitHub Issues**: For bug reports and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Documentation**: Check existing docs first

## Recognition

We appreciate all contributions! Contributors will be:

- **Listed in our README** and documentation
- **Mentioned in release notes** for significant contributions
- **Invited to join** our core contributor discussions

## Questions?

If you're unsure about anything or need guidance:

- **Open an issue** with the "question" label
- **Ask in GitHub Discussions**
- **Start with a small contribution** to get familiar

Thank you for helping make Colino better! 🙏