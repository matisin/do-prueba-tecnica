package migrator

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/matisin/pizzapp/pkgs/tester"
	_ "github.com/tursodatabase/go-libsql"
)

// RunMigrations performs a series of database migrations and verifies the results.
// It takes in a testing.T instance for logging purposes, a boolean indicating whether
// to run in inverse mode (for example, rolling back instead of migrating forward),
// the number of steps to perform during the first phase, and the number of steps to
// perform during the second phase. The function skips the test if firstSteps is less than 0.
func RunMigrations(t *testing.T, inverse bool, steps int, firstSteps int) {
	if firstSteps < 0 {
		t.Skip()
	}

	migrPath := tester.GenMigrationsPATH(t)
	stderr := &strings.Builder{}
	_, dbPath, db := tester.GenLibSqlDB(t)

	m := New(db, stderr, WithMigrationsDir(migrPath))

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("error closing conection after test")
		}
	})

	// Primera fase: ejecutamos las migraciones iniciales (setup)
	_, err := m.RunMigrations(firstSteps, false) // false = up
	if err != nil {
		t.Fatalf("failed initial migration: %v", err)
	}

	// Segunda fase: ejecutamos el caso de fuzzing
	_, err = m.RunMigrations(steps, inverse)
	if err != nil && err != ErrNegSteps {
		t.Fatalf("failed fuzzed migration: %v", err)
	}

	tester.AssertLibsqlDBNotEmtpy(t, dbPath)
}

// FuzzMigrator sets up a set of seed cases for fuzz testing to validate the RunMigrations function.
func FuzzMigrator(f *testing.F) {
	// Mantenemos los mismos casos semilla que son útiles para probar diferentes escenarios
	f.Add(false, 3, 0) // up, 3 steps
	f.Add(true, 1, 0)  // down, 1 step
	f.Add(false, 2, 3) // up, 2 steps con first steps
	f.Add(false, 0, 0) // up, 0 steps

	// La función principal de fuzzing
	f.Fuzz(RunMigrations)
}

// TestRunMigrationSim runs a simulation of running migrations with random seeds for each operation to ensure reproducibility.
func TestRunMigrationSim(t *testing.T) {
	migrPath := tester.GenMigrationsPATH(t)
	stderr := &strings.Builder{}
	_, dbPath, db := tester.GenLibSqlDB(t)

	m := New(db, stderr, WithMigrationsDir(migrPath))

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("error closing conection after test")
		}
	})

	const masterSeed int64 = 3151546024680003035
	const operations = 120

	masterRand := rand.New(rand.NewSource(masterSeed))

	seeds := make([]int64, operations)
	for i := 0; i < operations; i++ {
		seeds[i] = masterRand.Int63()
	}
	// There's one seed for operation because if we wanted to add more random generated data, if
	// there's one master seed for all operation and we generate a new random this will change the
	// result of all future operations, so with one seed per operation we ensure reproducibility of
	// the test.
	for i, seed := range seeds {
		rnd := rand.New(rand.NewSource(seed))

		inverse := rnd.Intn(2) == 0
		steps := rnd.Intn(8)

		// t.Logf("run operation %d with inverse %t and steps %d", i, inverse, steps)
		_, err := m.RunMigrations(steps, inverse) // false = up
		if err != nil {
			t.Fatalf("failed operation %d: %v", i, err)
		}
	}
	v, _ := m.Version()
	if v != 2 {
		t.Fatalf("expected final version to be 2, got %d", v)
	}
	tester.AssertLibsqlDBNotEmtpy(t, dbPath)
}
