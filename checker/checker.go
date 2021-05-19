package checker

import (
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/sharesies/infrastructure-integrity-lambda/checks"
)

type Checker struct {
	checks []checks.Check
	logger logrus.FieldLogger

	noticeMut sync.Mutex
	errorMut  sync.Mutex

	notices []checks.Notice
	errors  *multierror.Error
}

func NewChecker(logger logrus.FieldLogger, toCheck []checks.Check) *Checker {
	return &Checker{
		logger:  logger.WithField("package", "checker"),
		checks:  toCheck,
		notices: make([]checks.Notice, 0, len(toCheck)),
	}
}

func (c *Checker) CheckAll() ([]checks.Notice, error) {
	wg := sync.WaitGroup{}

	for _, check := range c.checks {
		wg.Add(1)
		go func(check checks.Check) {
			defer wg.Done()
			c.doCheck(check)
		}(check)
	}
	wg.Wait()

	return c.notices, c.errors.ErrorOrNil()
}

func (c *Checker) doCheck(check checks.Check) {
	c.logger.Infof("Running check %q...", check.Name())
	notices, err := check.Do()
	if err != nil {
		c.errorMut.Lock()
		defer c.errorMut.Unlock()
		multierror.Append(c.errors, errors.Wrapf(err, "running check %q", check.Name()))
	}

	c.noticeMut.Lock()
	defer c.noticeMut.Unlock()
	c.notices = append(c.notices, notices...)
}
