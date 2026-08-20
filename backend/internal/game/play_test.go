package game

import (
	"math/big"
	"reflect"
	"testing"
)

//func TestMultiplierFor(t *testing.T) {
//	first, err := multiplierFor(25, 3, 1)
//	if err != nil {
//		t.Fatalf("calculate first multiplier: %v", err)
//	}
//	if got := first.RatString(); got != "625/528" {
//		t.Fatalf("first multiplier = %s, want %s", got, "625/528")
//	}
//
//	second, err := multiplierFor(25, 3, 2)
//	if err != nil {
//		t.Fatalf("calculate second multiplier: %v", err)
//	}
//	if got := second.RatString(); got != "625/462" {
//		t.Fatalf("second multiplier = %s, want %s", got, "625/462")
//	}
//
//	payout, err := payoutFor(10, first)
//	if err != nil {
//		t.Fatalf("calculate payout: %v", err)
//	}
//	if payout != 11 {
//		t.Fatalf("payout = %d, want %d", payout, 11)
//	}
//}

func Test_multiplierFor(t *testing.T) {
	type args struct {
		gridSize    int
		mines       int
		openedCells int
	}
	tests := []struct {
		name    string
		args    args
		want    *big.Rat
		wantErr bool
	}{
		{
			name: "case 1.0",
			args: args{
				gridSize:    25,
				mines:       15,
				openedCells: 1,
			},
			want:    big.NewRat(24, 10),
			wantErr: false,
		},
		{
			name: "case 1.1",
			args: args{
				gridSize:    25,
				mines:       15,
				openedCells: 2,
			},
			want:    big.NewRat(576, 90),
			wantErr: false,
		},
		{
			name: "case 1.2",
			args: args{
				gridSize:    25,
				mines:       15,
				openedCells: 3,
			},
			want:    big.NewRat(13248, 720),
			wantErr: false,
		},
		{
			name: "36 cells first diamond",
			args: args{
				gridSize:    36,
				mines:       10,
				openedCells: 1,
			},
			want:    big.NewRat(432, 325),
			wantErr: false,
		},
		{
			name: "36 cells second diamond",
			args: args{
				gridSize:    36,
				mines:       10,
				openedCells: 2,
			},
			want:    big.NewRat(3024, 1625),
			wantErr: false,
		},
		{
			name: "49 cells first diamond",
			args: args{
				gridSize:    49,
				mines:       15,
				openedCells: 1,
			},
			want:    big.NewRat(588, 425),
			wantErr: false,
		},
		{
			name: "49 cells second diamond",
			args: args{
				gridSize:    49,
				mines:       15,
				openedCells: 2,
			},
			want:    big.NewRat(9408, 4675),
			wantErr: false,
		},
		{
			name: "64 cells first diamond",
			args: args{
				gridSize:    64,
				mines:       24,
				openedCells: 1,
			},
			want:    big.NewRat(192, 125),
			wantErr: false,
		},
		{
			name: "64 cells second diamond",
			args: args{
				gridSize:    64,
				mines:       24,
				openedCells: 2,
			},
			want:    big.NewRat(4032, 1625),
			wantErr: false,
		},
		{
			name: "single diamond",
			args: args{
				gridSize:    25,
				mines:       24,
				openedCells: 1,
			},
			want:    big.NewRat(24, 1),
			wantErr: false,
		},
		{
			name: "all diamonds opened",
			args: args{
				gridSize:    25,
				mines:       23,
				openedCells: 2,
			},
			want:    big.NewRat(288, 1),
			wantErr: false,
		},
		{
			name: "no diamonds opened",
			args: args{
				gridSize:    25,
				mines:       15,
				openedCells: 0,
			},
			wantErr: true,
		},
		{
			name: "negative opened cells",
			args: args{
				gridSize:    25,
				mines:       15,
				openedCells: -1,
			},
			wantErr: true,
		},
		{
			name: "more opened cells than diamonds",
			args: args{
				gridSize:    25,
				mines:       23,
				openedCells: 3,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := multiplierFor(tt.args.gridSize, tt.args.mines, tt.args.openedCells)
			if (err != nil) != tt.wantErr {
				t.Errorf("multiplierFor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("multiplierFor() got = %v, want %v", got, tt.want)
			}
		})
	}
}
